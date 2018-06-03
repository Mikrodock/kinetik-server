package docker

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/docker/go-connections/tlsconfig"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

type PortRange struct {
	Start int
	End   int
}

var stdlog, errlog *log.Logger

func Loggers(out *log.Logger, err *log.Logger) {
	stdlog = out
	errlog = err
}

func getClient() *client.Client {
	cli, err := client.NewClient("unix:///var/run/docker.sock", "v1.26", nil, nil)
	if err != nil {
		panic("Cannot connect to Docker!!")
	}
	return cli
}

func rangePortSet(start int, end int) nat.PortSet {
	portSet := make(nat.PortSet, end-start+1)
	var v struct{}
	for i := start; i <= end; i++ {
		port, _ := nat.NewPort("tcp", strconv.Itoa(i))
		portSet[port] = v
	}
	return portSet
}

func GetContainerIP(client *client.Client, id string, netName string) (string, error) {
	if client == nil {
		client = getClient()
	}
	ctx := context.Background()
	json, err := client.ContainerInspect(ctx, id)
	if err != nil {
		return "", err
	}
	for netMapName, netSetting := range json.NetworkSettings.Networks {
		stdlog.Printf("Map name %s. Asked name %s\n", netMapName, netName)
		if netMapName == netName {
			stdlog.Printf("IP is %s\n", netSetting.IPAddress)
			return netSetting.IPAddress, nil
		}
	}
	return "", nil
}

func RunContainer(imageName string, cmd []string, labels map[string]string, exposedPorts *PortRange, hostname string, dnsServers []string) (string, error) {
	client := getClient()

	ctx := context.Background()

	reader, err := client.ImagePull(ctx, imageName, types.ImagePullOptions{})
	ioutil.ReadAll(reader)

	if err != nil {
		return "", err
	}

	containerConfig := &container.Config{
		Image:    imageName,
		Hostname: hostname,
		Labels:   labels,
	}

	if len(cmd) != 0 {
		containerConfig.Cmd = cmd
	}

	if exposedPorts != nil {
		ports := rangePortSet(exposedPorts.Start, exposedPorts.End)
		containerConfig.ExposedPorts = ports
	}

	hostConfig := &container.HostConfig{}
	if len(dnsServers) != 0 {
		hostConfig.DNS = dnsServers
	}

	mikroverlayConfig := &network.EndpointSettings{
		IPAMConfig: nil,
		Links:      nil,
	}

	endsConfig := make(map[string]*network.EndpointSettings)
	endsConfig["mikroverlay"] = mikroverlayConfig

	netConfig := &network.NetworkingConfig{
		EndpointsConfig: endsConfig,
	}

	cnt, err := client.ContainerCreate(ctx, containerConfig, hostConfig, netConfig, hostname)

	if err != nil {
		return "", err
	}

	cntID := cnt.ID

	err = client.ContainerStart(ctx, cntID, types.ContainerStartOptions{})

	if err != nil {
		return "", err
	}

	return cntID, nil
}

func RunContainerFromConfig(client *client.Client, config *types.ContainerCreateConfig) (string, error) {
	if client == nil {
		client = getClient()
	}

	ctx := context.Background()

	reader, err := client.ImagePull(ctx, config.Config.Image, types.ImagePullOptions{})

	if err != nil {
		return "", err
	}

	ioutil.ReadAll(reader)

	if err != nil {
		return "", err
	}

	cnt, err := client.ContainerCreate(ctx, config.Config, config.HostConfig, config.NetworkingConfig, config.Name)

	if err != nil {
		return "", err
	}

	cntID := cnt.ID

	err = client.ContainerStart(ctx, cntID, types.ContainerStartOptions{})

	if err != nil {
		return "", err
	}

	return cntID, nil
}

func GetRemoteClient(nodeIP string) (*client.Client, error) {
	options := tlsconfig.Options{
		CAFile:             filepath.Join("/etc/docker", "ca.cert"),
		CertFile:           filepath.Join("/etc/docker", "cert.pem"),
		KeyFile:            filepath.Join("/etc/docker", "key.pem"),
		InsecureSkipVerify: false,
	}
	tlsc, err := tlsconfig.Client(options)
	if err != nil {
		return nil, err
	}

	httpclient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsc,
		},
	}

	headers := make(map[string]string)

	cli, err := client.NewClient("tcp://"+nodeIP+":2376", "v1.26", httpclient, headers)

	return cli, err

}

func WaitForMikroverlayNetwork() {
	client := getClient()
	tries := 0
	ctx := context.Background()
	for tries < 20 {
		netlist, err := client.NetworkList(ctx, types.NetworkListOptions{})
		if err == nil {
			for _, net := range netlist {
				if net.Name == "mikroverlay" {
					return
				}
			}
		}
		time.Sleep(10 * time.Second)
		tries++
	}

}
