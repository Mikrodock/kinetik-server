package services

import (
	"encoding/json"
	"kinetik-server/compose"
	"kinetik-server/control"
	"kinetik-server/data"
	"kinetik-server/docker"
	"kinetik-server/iptables"
	"kinetik-server/models/internals"
	"kinetik-server/models/v2"
	"kinetik-server/scheduler"
	"net/http"
	"strconv"
	"strings"

	composeTypes "github.com/docker/cli/cli/compose/types"
	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"

	"github.com/gorilla/mux"
)

func GetServices(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(data.GetDB().GetServices())
}

func AddService(w http.ResponseWriter, r *http.Request) {

	var srvCreateReq v2.ServiceCreationRequest

	sch := scheduler.GetScheduler()

	err := json.NewDecoder(r.Body).Decode(&srvCreateReq)

	if err != nil {
		http.Error(w, "Cannot decode body : "+err.Error(), 500)
		return
	}

	config, err := compose.LoadYAMLWithEnv([]byte(srvCreateReq.DockerComposeContent), nil)

	if err != nil {
		http.Error(w, "Cannot decode YAML : "+err.Error(), 500)
	}

	labels := make(map[string]string)
	srvContainerConfig := make(map[string]*dockerTypes.ContainerCreateConfig)
	srvReplica := make(map[string]uint64)
	srvPorts := make(map[string][]composeTypes.ServicePortConfig)
	srvConstraints := make(map[string]*composeTypes.Resource)

	debugMap := make(map[string]interface{})

	mikroverlayConfig := &network.EndpointSettings{
		IPAMConfig: nil,
		Links:      nil,
	}

	endsConfig := make(map[string]*network.EndpointSettings)
	endsConfig["mikroverlay"] = mikroverlayConfig

	for _, srv := range config.Services {
		contConfig, _ := compose.ConvertServiceToContainer(&srv)
		contConfig.HostConfig.DNS = []string{data.GetDB().GetConfig().DNSIP}
		labels["be.mikrodock.stack"] = srvCreateReq.StackName
		labels["be.mikrodock.service"] = srv.Name
		contConfig.Config.Labels = labels
		contConfig.HostConfig.DNSSearch = []string{srvCreateReq.StackName + ".mikrodock"}
		contConfig.NetworkingConfig = &network.NetworkingConfig{
			EndpointsConfig: endsConfig,
		}
		srvContainerConfig[srv.Name] = contConfig
		srvConstraints[srv.Name] = srv.Deploy.Resources.Reservations

		srvPorts[srv.Name] = srv.Ports

		if srv.Deploy.Replicas == nil {
			srvReplica[srv.Name] = 1
		} else {
			srvReplica[srv.Name] = *srv.Deploy.Replicas
		}

	}

	for srvName, config := range srvContainerConfig {
		srvIPs := make([]string, srvReplica[srvName])
		srvIPWeights := make([]control.IPWithWeight, 0)
		for i := 0; i < int(srvReplica[srvName]); i++ {

			nodeIP, _ := sch.SelectWithConstraints(srvConstraints[srvName])
			client, err := docker.GetRemoteClient(nodeIP)
			if err != nil {
				http.Error(w, "Cannot get remote client"+err.Error(), 500)
			}

			id, err := docker.RunContainerFromConfig(client, config)
			if err != nil {
				http.Error(w, "Cannot run service "+srvName+"   "+err.Error(), 500)
			}
			ip, err := docker.GetContainerIP(client, id, "mikroverlay")
			if err != nil {
				http.Error(w, "Cannot run service "+srvName+"   "+err.Error(), 500)
			}
			srvIPs = append(srvIPs, ip)
			srvIPWeights = append(srvIPWeights, control.IPWithWeight{
				IP:     ip,
				Weight: 10,
			})
		}
		control.AddToDNS(srvName, srvCreateReq.StackName, srvIPWeights)

		proxyIPParts := strings.Split(data.GetDB().GetConfig().ProxyIP, ".")
		lastpart := proxyIPParts[len(proxyIPParts)-1]
		proxyIP := "172.18.0." + lastpart

		for _, portConfig := range srvPorts[srvName] {
			control.AddToProxy(srvName, srvCreateReq.StackName, int(portConfig.Target), int(portConfig.Published))

			iptables.NewLinkPort(proxyIP, int(portConfig.Published), int(portConfig.Target))
		}
	}

	debugMap["services"] = srvContainerConfig
	debugMap["ports"] = srvPorts

	json.NewEncoder(w).Encode(debugMap)

}

func DeleteService(w http.ResponseWriter, r *http.Request) {

	// TODO : Delete DO LB

	params := mux.Vars(r)
	id := params["id"]
	idInt, err := strconv.Atoi(id)
	err = data.GetDB().DeleteService(idInt)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(internals.ErrorMessage{
			Message: err.Error(),
		})
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func ScaleUp(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]
	idInt, err := strconv.Atoi(id)
	err = data.GetDB().DeleteService(idInt)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(internals.ErrorMessage{
			Message: err.Error(),
		})
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func ScaleDown(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]
	idInt, err := strconv.Atoi(id)
	err = data.GetDB().DeleteService(idInt)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(internals.ErrorMessage{
			Message: err.Error(),
		})
	} else {
		w.WriteHeader(http.StatusOK)
	}
}
