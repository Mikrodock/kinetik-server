package control

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"kinetik-server/data"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

var netClient = &http.Client{
	Timeout: time.Second * 10,
}

type IPWithWeight struct {
	IP     string
	Weight int
}

func AddToDNS(serviceName string, stackName string, ips []IPWithWeight) {
	var wg sync.WaitGroup

	wg.Add(len(ips))

	dnsIPParts := strings.Split(data.GetDB().GetConfig().DNSIP, ".")
	lastpart := dnsIPParts[len(dnsIPParts)-1]
	dnsIP := "172.18.0." + lastpart

	for _, ip := range ips {
		go func(registerName string, ip IPWithWeight, wgI *sync.WaitGroup) {

			res, err := netClient.Post("http://"+dnsIP+":8080/api/domains/"+registerName, "text/plain", bytes.NewBufferString(ip.IP+" "+strconv.Itoa(ip.Weight)))
			if err != nil {
				panic(err)
			}
			defer res.Body.Close()
			ioutil.ReadAll(res.Body)

			wgI.Done()
		}(
			serviceName+"."+stackName+".mikrodock",
			ip,
			&wg,
		)
	}

	wg.Wait()
}

type ServiceCreationRequest struct {
	ServiceName  string `json:"service_name"`
	StackName    string `json:"stack_name"`
	PublicPort   int    `json:"public_port"`
	InternalPort int    `json:"internal_port"`
}

func AddToProxy(serviceName string, stackName string, internalPort int, publicPort int) {
	srvCrReq := ServiceCreationRequest{
		ServiceName:  serviceName,
		StackName:    stackName,
		PublicPort:   publicPort,
		InternalPort: internalPort,
	}

	jsonValue, _ := json.Marshal(srvCrReq)
	jsonBuffer := bytes.NewBuffer(jsonValue)

	proxyIPParts := strings.Split(data.GetDB().GetConfig().ProxyIP, ".")
	lastpart := proxyIPParts[len(proxyIPParts)-1]
	proxyIP := "172.18.0." + lastpart

	res, _ := netClient.Post("http://"+proxyIP+":10512/services/", "application/json", jsonBuffer)
	defer res.Body.Close()
	ioutil.ReadAll(res.Body)

}

func RemoveFromProxy(serviceName, stackName string) {
	emptyBuffer := bytes.NewBuffer([]byte{})

	proxyIPParts := strings.Split(data.GetDB().GetConfig().ProxyIP, ".")
	lastpart := proxyIPParts[len(proxyIPParts)-1]
	proxyIP := "172.18.0." + lastpart

	res, _ := netClient.Post("http://"+proxyIP+":10512/services/"+stackName+"/"+serviceName, "application/json", emptyBuffer)
	defer res.Body.Close()
	ioutil.ReadAll(res.Body)
}

func RemoveFromDNS(serviceName, stackName, containerIP string) {
	dnsIPParts := strings.Split(data.GetDB().GetConfig().DNSIP, ".")
	lastpart := dnsIPParts[len(dnsIPParts)-1]
	dnsIP := "172.18.0." + lastpart

	req, _ := http.NewRequest("DELETE", "http://"+dnsIP+":8080/api/domains/"+serviceName+"."+stackName+".mikrodock", bytes.NewBufferString(containerIP))

	res, err := netClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	ioutil.ReadAll(res.Body)
}
