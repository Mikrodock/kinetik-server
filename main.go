package main

import (
	"fmt"
	"kinetik-server/boltdb"
	"kinetik-server/data"
	"kinetik-server/docker"
	"kinetik-server/handlers/instances"
	"kinetik-server/handlers/nodes"
	"kinetik-server/handlers/services"
	"kinetik-server/logger"
	"kinetik-server/models"
	"kinetik-server/models/internals"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/takama/daemon"
)

var stdlog, errlog *log.Logger

func init() {
	logFile, _ := os.OpenFile("/var/log/kinetik.out", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	errFile, _ := os.OpenFile("/var/log/kinetik.err", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	stdlog = log.New(logFile, "", log.Ldate|log.Ltime)
	errlog = log.New(errFile, "", log.Ldate|log.Ltime)
	logger.StdLog = stdlog
	logger.ErrLog = errlog
}

func main() {

	rand.Seed(time.Now().Unix())

	srv, err := daemon.New("kinetik-server", "Server that handles request from nodes")
	if err != nil {
		errlog.Println("Error: ", err)
		os.Exit(1)
	}
	service := &Service{srv}

	service.Run()

}

type Service struct {
	daemon.Daemon
}

func intToPointer(a int) *int {
	return &a
}

func floatToPointer(f float32) *float32 {
	return &f
}

func stateToPointer(s models.StateValue) *models.StateValue {
	return &s
}

func retry(attempts int, sleep time.Duration, fn func() (interface{}, error)) (result interface{}, err error) {
	var res interface{}
	for i := 0; ; i++ {
		res, err := fn()
		if err == nil {
			return res, nil
		}

		if i >= (attempts - 1) {
			break
		}

		time.Sleep(sleep)

		log.Println("retrying after error:", err)
	}
	return res, fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}

func (service *Service) Run() (string, error) {

	usage := "Usage: kinetik-server install | remove | start | stop | status"

	// if received any kind of command, do it
	if len(os.Args) > 1 {
		command := os.Args[1]
		switch command {
		case "install":
			return service.Install()
		case "remove":
			return service.Remove()
		case "start":
			return service.Start()
		case "stop":
			return service.Stop()
		case "status":
			return service.Status()
		default:
			return usage, nil
		}
	}

	docker.Loggers(stdlog, errlog)

	if boltdb.IsFirstRun() {

		for {
			if _, err := os.Stat("/var/run/docker.sock"); err != nil {
				stdlog.Println("Docker is not ready yet... Waiting 10 seconds")
				time.Sleep(10 * time.Second)
				_, err = os.Stat("/var/run/docker.sock")
			} else {
				stdlog.Println("Waiting 60 seconds for Docker, just to be sure...")
				time.Sleep(60 * time.Second)
				break
			}

		}

		docker.WaitForMikroverlayNetwork()

		stdlog.Println("Starting Kinetik management containers... (1/2)")
		mikrodnsLabels := make(map[string]string)
		mikrodnsLabels["be.mikrodock.management"] = "dns"
		dnsID, err := retry(3, 30*time.Second, func() (interface{}, error) {
			return docker.RunContainer("izanagi1995/mikrodns:latest", []string{}, mikrodnsLabels, nil, "dns", []string{})
		})
		if err != nil {
			errlog.Fatalln("Cannot start dns container : " + err.Error())
		}
		dnsIP, err := docker.GetContainerIP(nil, dnsID.(string), "mikroverlay")

		if err != nil {
			errlog.Fatalln("Cannot get dns ip : " + err.Error())
		}

		stdlog.Println("Starting Kinetik management containers... (2/2)")
		mikrodnsLabels["be.mikrodock.management"] = "proxy"
		proxyID, err := retry(3, 30*time.Second, func() (interface{}, error) {
			return docker.RunContainer("izanagi1995/mikroproxy:latest", []string{"--dns", dnsIP}, mikrodnsLabels, &docker.PortRange{
				Start: 80,
				End:   8080,
			}, "proxy", []string{dnsIP})
		})
		if err != nil {
			errlog.Fatalln("Cannot start proxy container : " + err.Error())
		}
		stdlog.Println("Starting Kinetik management containers... Done")
		proxyIP, err := docker.GetContainerIP(nil, proxyID.(string), "mikroverlay")

		if err != nil {
			errlog.Fatalln("Cannot get proxy ip : " + err.Error())
		}

		config := &internals.Config{
			DNSID:        dnsID.(string),
			DNSIP:        dnsIP,
			ProxyID:      proxyID.(string),
			ProxyIP:      proxyIP,
			PortsBinding: []int{},
		}

		err = data.GetDB().SetConfig(config)

		if err != nil {
			errlog.Fatalln("Cannot save config : " + err.Error())
		}

		stdlog.Println("Config saved...")
	} else {
		stdlog.Printf("%#v\n", data.GetDB().GetConfig())
	}

	router := mux.NewRouter()
	ConfigureRouter(router)

	srv := &http.Server{Addr: ":10513"}

	srv.Handler = router

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			// cannot panic, because this probably is an intentional close
			stdlog.Printf("Httpserver: ListenAndServe() error: %s", err)
		}
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGINT)

	for {
		select {
		case killSignal := <-interrupt:
			srv.Shutdown(nil)
			stdlog.Println("Kinetik-Server - Bye - " + killSignal.String())
		}
	}

}

func ConfigureRouter(router *mux.Router) {
	router.HandleFunc("/services", services.GetServices).Methods("GET")
	router.HandleFunc("/services", services.AddService).Methods("POST")

	router.HandleFunc("/services/{stack}/{service}", services.DeleteService).Methods("DELETE")
	router.HandleFunc("/services/{stack}/{service}/scale/up", services.ScaleUp).Methods("POST")
	router.HandleFunc("/services/{stack}/{service}/scale/down", services.ScaleDown).Methods("POST")

	router.HandleFunc("/nodes/{id}", nodes.UpdateNode).Methods("POST")

	router.HandleFunc("/instances", instances.GetInstances).Methods("GET")
	router.HandleFunc("/instances/{id}", instances.DeleteInstance).Methods("DELETE")
	router.HandleFunc("/instances/{id}", instances.UpdateMetrics).Methods("PUT")
}
