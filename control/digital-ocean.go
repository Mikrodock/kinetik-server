package control

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"kinetik-server/logger"
	"log"
	"net"
	"os"
	"time"

	"github.com/digitalocean/godo"
	"github.com/tmc/scp"

	"golang.org/x/crypto/ssh"
	"golang.org/x/oauth2"
)

type DockerClusterOptions struct {
	ClusterStoreAddress string
	AdvertiseAddress    string
	CAPath              string
	CertPath            string
	KeyPath             string
}

func (d *DockerClusterOptions) String() string {
	var sb bytes.Buffer
	if len(d.ClusterStoreAddress) != 0 {
		sb.WriteString("--cluster-store=consul://" + d.ClusterStoreAddress + "\n")
	}
	if len(d.AdvertiseAddress) != 0 {
		sb.WriteString("--cluster-advertise=" + d.AdvertiseAddress + "\n")
	}
	if len(d.CAPath) != 0 {
		sb.WriteString("--cluster-store-opt kv.cacertfile=" + d.CAPath + "\n")
	}
	if len(d.CertPath) != 0 {
		sb.WriteString("--cluster-store-opt kv.certfile=" + d.CertPath + "\n")
	}
	if len(d.KeyPath) != 0 {
		sb.WriteString("--cluster-store-opt kv.keyfile=" + d.KeyPath + "\n")
	}
	return sb.String()
}

type KlerkCreationOption struct {
	Token             string
	SSHKeyFingerprint string
	Name              string
	Size              string
	Region            string
}

type PartikleConfig struct {
	Name      string
	IP        string
	SSHPort   int
	SSHUser   string
	MachineID int
	IsMaster  bool
}

var kLog *log.Logger

func CreateKlerk(options *KlerkCreationOption) (*PartikleConfig, error) {

	kLog = logger.NewLogger(options.Name + ".log")

	// STEPS TO GO :
	// CREATE A NEW MACHINE ON DO

	tSource := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: options.Token,
	})
	oauthClient := oauth2.NewClient(context.Background(), tSource)
	client := godo.NewClient(oauthClient)

	createRequest := godo.DropletCreateRequest{
		Name:   options.Name,
		Size:   options.Size,
		Region: options.Region,
		Image: godo.DropletCreateImage{
			Slug: "docker",
		},
		SSHKeys: []godo.DropletCreateSSHKey{godo.DropletCreateSSHKey{
			Fingerprint: options.SSHKeyFingerprint,
		}},
	}

	dropRes, raw, err := client.Droplets.Create(context.Background(), &createRequest)

	if err != nil {
		rawBody, _ := ioutil.ReadAll(raw.Body)
		kLog.Println("Droplet creation failed : ", err.Error(), ". Got body:", string(rawBody))
		return nil, err
	}

	// Wait active
	drop, _, _ := client.Droplets.Get(context.Background(), dropRes.ID)
	for drop.Status != "active" {
		kLog.Println("Droplet in state", drop.Status, ". Waiting 10 seconds")
		time.Sleep(10 * time.Second)
		drop, _, _ = client.Droplets.Get(context.Background(), dropRes.ID)
	}

	time.Sleep(10 * time.Second)

	publicv4, _ := drop.PublicIPv4()

	for publicv4 == "" {
		logger.StdLog.Printf("Could not get IPv4 of %d, retrying", drop.ID)
		time.Sleep(1 * time.Second)
		publicv4, err = drop.PublicIPv4()
	}

	kLog.Println("Droplet created. IP", publicv4)

	logger.StdLog.Printf("Node %s (%d) is now running with IP %s\n", options.Name, dropRes.ID, publicv4)

	if err != nil {
		return nil, err
	}

	//SSH PATH
	sshPath := "/root/.ssh/id_rsa"

	// PRIVISION ENV VARS
	envVars := make(map[string]string)
	envVars["CONSUL_IP"] = os.Getenv("CONSUL_IP")
	myIP, _ := getMyIP()
	envVars["KINETIK_MASTER"] = myIP + ":10513"

	sshClient, err := connectSSH(sshPath, publicv4)
	if err != nil {
		kLog.Println("SSH error :", err.Error())
		return nil, err
	}

	kLog.Println("Droplet SSH ready")

	for key, value := range envVars {
		SSHCommand(sshClient, fmt.Sprintf("echo 'export %s=%s' >> ~/.env", key, value))
	}
	SSHCommand(sshClient, "echo 'source ~/.env'")

	// CONFIGURE DOCKER
	_, _, err = SSHCommand(sshClient, "service docker stop")
	if err != nil {
		return nil, err
	}

	dockerConfig := &DockerClusterOptions{
		AdvertiseAddress:    publicv4 + ":2376",
		ClusterStoreAddress: os.Getenv("CONSUL_IP"),
		CAPath:              "/etc/docker/kv-ca.cert",
		CertPath:            "/etc/docker/kv-cert.pem",
		KeyPath:             "/etc/docker/kv-key.pem",
	}

	dockerConf := `DOCKER_OPTS='
-H tcp://0.0.0.0:2376
-H unix:///var/run/docker.sock
--tlsverify
--tlscacert /etc/docker/ca.cert
--tlscert /etc/docker/cert.pem
--tlskey /etc/docker/key.pem
` + dockerConfig.String() + `'`

	confCmd := fmt.Sprintf("printf %%s \"%s\" | tee /etc/default/docker", dockerConf)

	_, _, err = SSHCommand(sshClient, confCmd)
	if err != nil {
		return nil, err
	}

	// DONT RESTART DOCKER YET, WE NEED TO SEND THE CERTIFICATE FROM THE CLIENT

	// _, _, err = SSHCommand(sshClient, "service docker start")
	// if err != nil {
	// 	return nil, err
	// }

	// INSTALL KINETIK-CLIENT

	_, _, err = SSHCommand(sshClient, "wget https://nsurleraux.be/kinetik-client -O /usr/bin/kinetik-client")
	if err != nil {
		return nil, err
	}

	_, _, err = SSHCommand(sshClient, "chmod +x /usr/bin/kinetik-client")
	if err != nil {
		return nil, err
	}
	_, _, err = SSHCommand(sshClient, "kinetik-client install")
	if err != nil {
		return nil, err
	}
	_, _, err = SSHCommand(sshClient, "kinetik-client start")
	if err != nil {
		return nil, err
	}

	sshClient.Close()

	logger.StdLog.Printf("Node %s (%d) is now ready to receive certs\n", options.Name, dropRes.ID)
	kLog.Println("======== END " + options.Name + " ========")

	cfg := &PartikleConfig{
		IP:        publicv4,
		IsMaster:  false,
		MachineID: drop.ID,
		Name:      options.Name,
		SSHPort:   22,
		SSHUser:   "root",
	}

	return cfg, nil

}

func StartDocker(ip string) error {
	sshPath := "/root/.ssh/id_rsa"
	sshClient, err := connectSSH(sshPath, ip)
	if err != nil {
		kLog.Println("SSH error :", err.Error())
		return err
	}
	_, _, err = SSHCommand(sshClient, "service docker start")
	if err != nil {
		kLog.Println("SSH error :", err.Error())
		return err
	}
	return nil
}

func copyFile(sshClient *ssh.Client, source, destination string) error {
	session, err := sshClient.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	return scp.CopyPath(source, destination, session)
}

func getMyIP() (string, error) {
	netIf, err := net.InterfaceByName("eth0")
	if err != nil {
		return "", err
	}

	addrs, err := netIf.Addrs()
	if err != nil {
		return "", err
	}

	var ip net.IP

	for _, addr := range addrs {
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip != nil && ip.To4() != nil {
			break
		}
	}

	if ip != nil {
		return ip.String(), nil
	} else {
		return "", errors.New("Cannot resolve IP of eth0 interface")
	}

}

func publicKeyFile(file string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(key)
}

func connectSSH(sshKeyPath, ip string) (*ssh.Client, error) {

	logger.StdLog.Printf("Opening SSH connection to %s\n", ip)

	sshConfig := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			publicKeyFile(sshKeyPath),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	retryCount := 1
	var _err error
	for retryCount < 100 {
		kLog.Printf("SSH : retry n°%d\n", retryCount)
		var err error
		client, err := ssh.Dial("tcp", ip+":22", sshConfig)
		if err != nil {
			time.Sleep(10 * time.Second)
			retryCount++
			_err = err
		} else {
			_err = nil
			return client, _err
		}
	}

	if _err != nil {
		kLog.Printf("SSH : cannot connect after 100 retries WTF! Quit...")
	}

	return nil, _err
}

func SSHCommand(client *ssh.Client, cmd string) (string, string, error) {
	sess, err := client.NewSession()
	if err != nil {
		kLog.Println("Cannot open SSH session", err.Error())
		return "", "", err
	}
	defer sess.Close()
	var stdoutBuf bytes.Buffer
	sess.Stdout = &stdoutBuf
	var stderrBuf bytes.Buffer
	sess.Stderr = &stderrBuf

	err = sess.Run(cmd)

	if err != nil {
		kLog.Printf("Error while running command %s\n Err : %s\nSSHOut : %s\nSSHErr : %s\n", cmd, err.Error(), stdoutBuf.String(), stderrBuf.String())
	}

	return stdoutBuf.String(), stderrBuf.String(), err
}

func LoadPrivateKey(file string) (ssh.Signer, error) {
	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	key, err := ssh.ParsePrivateKey(buf)
	if err != nil {
		return nil, err
	}

	return key, nil
}

func ComputePublicFingerprint(keyPath string) string {
	sig, _ := LoadPrivateKey(keyPath)
	return ssh.FingerprintLegacyMD5(sig.PublicKey())
}
