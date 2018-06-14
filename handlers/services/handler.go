package services

import (
	"context"
	"encoding/json"
	"kinetik-server/compose"
	"kinetik-server/control"
	"kinetik-server/data"
	"kinetik-server/docker"
	"kinetik-server/iptables"
	"kinetik-server/logger"
	"kinetik-server/models"
	"kinetik-server/models/v2"
	"kinetik-server/scheduler"
	"math/rand"
	"net/http"
	"strings"
	"time"

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

	workGraph := make(models.Graph, len(config.Services))

	debugMap := make(map[string]interface{})

	mikroverlayConfig := &network.EndpointSettings{
		IPAMConfig: nil,
		Links:      nil,
	}

	endsConfig := make(map[string]*network.EndpointSettings)
	endsConfig["mikroverlay"] = mikroverlayConfig

	for i, srv := range config.Services {

		workGraph[i] = models.NewDepNode(srv.Name, srv.DependsOn)

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

	depGraph := workGraph.Resolve()

	for _, node := range depGraph {
		srvName := node.Name
		config := srvContainerConfig[srvName]
		serviceModel := models.NewService(srvCreateReq.StackName, srvName, config)
		serviceModel.Constraints = srvConstraints[srvName]
		serviceModel.Ports = srvPorts[srvName]

		srvIPs := make([]string, srvReplica[srvName])
		srvIPWeights := make([]control.IPWithWeight, 0)
		for i := 0; i < int(srvReplica[srvName]); i++ {

			nodeIP, _ := sch.SelectWithConstraints(srvConstraints[srvName])

			// TODO : handle error

			client, err := docker.GetRemoteClient(nodeIP)
			if err != nil {
				http.Error(w, "Cannot get remote client"+err.Error(), 500)
			}

			id, err := docker.RunContainerFromConfig(client, config)
			if err != nil {
				http.Error(w, "Cannot run service "+srvName+"   "+err.Error(), 500)
			}

			serviceModel.AddInstance(&models.Instance{
				ContainerID: id,
				NodeID:      nodeIP,
			})

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

		data.GetDB().AddService(serviceModel)

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

func timeoutSeconds(n int) *time.Duration {
	a := time.Duration(n) * time.Second
	return &a
}

func DeleteService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	stack := vars["stack"]
	service := vars["service"]

	id := stack + "/" + service
	srv := data.GetDB().GetService(id)

	if srv == nil {
		http.Error(w, "No service "+id, 404)
		return
	}

	ctx := context.Background()

	for _, inst := range srv.Instances {
		node := inst.NodeID
		remoteClient, _ := docker.GetRemoteClient(node)
		_ = remoteClient.ContainerStop(ctx, inst.ContainerID, timeoutSeconds(5))
		_ = remoteClient.ContainerRemove(ctx, inst.ContainerID, dockerTypes.ContainerRemoveOptions{
			Force: true,
		})
	}

	w.WriteHeader(200)

}

func ScaleUp(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	stack := params["stack"]
	service := params["service"]
	srv := data.GetDB().GetService(stack + "/" + service)
	if srv == nil {
		http.Error(w, "Service not found "+stack+"/"+service, 404)
		return
	}
	nodeIP, err := scheduler.GetScheduler().SelectWithConstraints(srv.Constraints)
	if err != nil {
		http.Error(w, "Cannot get Node", 500)
	}
	dockerClient, err := docker.GetRemoteClient(nodeIP)
	if err != nil {
		http.Error(w, "Cannot get remote client", 500)
	}

	id, err := docker.RunContainerFromConfig(dockerClient, srv.ContainerConfig)
	if err != nil {
		http.Error(w, "Cannot run service "+srv.ServiceName+" : "+err.Error(), 500)
	}

	ip, err := docker.GetContainerIP(dockerClient, id, "mikroverlay")
	if err != nil {
		http.Error(w, "Cannot run service "+srv.ServiceName+" : "+err.Error(), 500)
	}

	srv.AddInstance(&models.Instance{
		ContainerID: id,
		NodeID:      nodeIP,
	})

	control.AddToDNS(srv.ServiceName, srv.StackName, []control.IPWithWeight{control.IPWithWeight{
		IP:     ip,
		Weight: 10,
	}})

	proxyIPParts := strings.Split(data.GetDB().GetConfig().ProxyIP, ".")
	lastpart := proxyIPParts[len(proxyIPParts)-1]
	proxyIP := "172.18.0." + lastpart

	for _, portConfig := range srv.Ports {
		control.AddToProxy(srv.ServiceName, srv.StackName, int(portConfig.Target), int(portConfig.Published))

		iptables.NewLinkPort(proxyIP, int(portConfig.Published), int(portConfig.Target))
	}

	data.GetDB().AddService(srv)
}

func ScaleDown(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	stack := params["stack"]
	service := params["service"]
	srv := data.GetDB().GetService(stack + "/" + service)
	if srv == nil {
		http.Error(w, "Service not found "+stack+"/"+service, 404)
		return
	}

	idx := rand.Intn(len(srv.Instances))
	inst := srv.Instances[idx]

	ctx := context.Background()

	node := inst.NodeID
	remoteClient, _ := docker.GetRemoteClient(node)
	ip, _ := docker.GetContainerIP(remoteClient, inst.ContainerID, "mikroverlay")
	logger.StdLog.Printf("Scaling down %s/%s : Container %s down with IP %s\n", stack, service, node, ip)
	control.RemoveFromDNS(service, stack, ip)
	_ = remoteClient.ContainerStop(ctx, inst.ContainerID, timeoutSeconds(5))
	_ = remoteClient.ContainerRemove(ctx, inst.ContainerID, dockerTypes.ContainerRemoveOptions{
		Force: true,
	})

	srv.Instances = append(srv.Instances[:idx], srv.Instances[idx+1:]...)

	data.GetDB().AddService(srv)

}
