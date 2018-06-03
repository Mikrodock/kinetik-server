package fakedb

import (
	"errors"
	"kinetik-server/debug"
	"kinetik-server/models"
	"time"

	"github.com/c2h5oh/datasize"
)

type FakeDB struct {
	Nodes     []*models.Node
	Services  []*models.Service
	Instances []*models.Instance
}

var cpuRegMetrics = []*models.MetricDescriptor{
	&models.MetricDescriptor{
		Name: "cpu",
		Breakpoints: []*models.MetricBreakpoint{
			&models.MetricBreakpoint{
				State: debug.StateToPointer(models.Ok),
				Value: debug.FloatToPointer(0),
			},
		},
	},
}

var cpuMetrics = []*models.MetricValue{
	&models.MetricValue{
		Name:  "cpu",
		Value: debug.FloatToPointer(0),
	},
}

func NewFakeDB() *FakeDB {

	f := &FakeDB{
		Nodes:     make([]*models.Node, 0),
		Instances: make([]*models.Instance, 0),
		Services:  make([]*models.Service, 0),
	}

	n1 := &models.Node{
		Name:              "Node1",
		IP:                "1.1.1.1",
		State:             debug.StateToPointer(models.Ok),
		RegisteredMetrics: cpuRegMetrics,
		Metrics:           cpuMetrics,
	}

	s1 := &models.Service{
		Name:              "TestService",
		CreationDate:      time.Now(),
		UpdateDate:        time.Now(),
		GlobalState:       debug.StateToPointer(models.Ok),
		ID:                debug.IntToPointer(1),
		Nodes:             []*models.Node{n1},
		RegisteredMetrics: cpuRegMetrics,
		Metrics:           cpuMetrics,
		Reservation: models.Reservation{
			MemReservation:  100 * datasize.MB,
			CPUReservation:  100,
			DiskReservation: 3 * datasize.GB,
		},
	}

	i1 := &models.Instance{
		ID:                debug.IntToPointer(1),
		ServiceID:         debug.IntToPointer(1),
		NodeID:            debug.IntToPointer(1),
		Metrics:           cpuMetrics,
		RegisteredMetrics: cpuRegMetrics,
		State:             debug.StateToPointer(models.Ok),
		Timeout:           debug.IntToPointer(0),
	}

	n1.Instances = []*models.Instance{i1}
	s1.Instances = []*models.Instance{i1}

	f.Instances = []*models.Instance{i1}
	f.Nodes = []*models.Node{n1}
	f.Services = []*models.Service{s1}

	return f

}

func (f *FakeDB) GetNodes() []*models.Node {
	return f.Nodes
}

func (f *FakeDB) GetServices() []*models.Service {
	return f.Services
}

func (f *FakeDB) GetInstances() []*models.Instance {
	return f.Instances
}

func (f *FakeDB) AddNode(node *models.Node) error {
	node.ID = debug.IntToPointer(len(f.Nodes) + 1)
	f.Nodes = append(f.Nodes, node)
	return nil
}

func (f *FakeDB) DeleteNode(nodeID int) error {
	for i, node := range f.Nodes {
		if *node.ID == nodeID {
			f.Nodes = append(f.Nodes[:i], f.Nodes[i+1:]...)
			return nil
		}
	}
	return errors.New("Cannot find node")
}

func (f *FakeDB) AddService(service *models.Service) error {
	service.ID = debug.IntToPointer(len(f.Services) + 1)
	service.CreationDate = time.Now()
	service.UpdateDate = time.Now().Add(10 * time.Second)
	service.Metrics = cpuMetrics
	service.GlobalState = debug.StateToPointer(models.Ok)
	f.Services = append(f.Services, service)
	return nil
}

func (f *FakeDB) DeleteService(serviceID int) error {
	for i, service := range f.Services {
		if *service.ID == serviceID {
			f.Services = append(f.Services[:i], f.Services[i+1:]...)
			return nil
		}
	}
	return errors.New("Cannot find service")
}

func (f *FakeDB) AddInstance(instance *models.Instance) error {
	instance.ID = debug.IntToPointer(len(f.Instances) + 1)
	f.Instances = append(f.Instances, instance)
	return nil
}

func (f *FakeDB) DeleteInstance(instanceID int) error {
	for i, instance := range f.Instances {
		if *instance.ID == instanceID {
			f.Instances = append(f.Instances[:i], f.Instances[i+1:]...)
			return nil
		}
	}
	return errors.New("Cannot find instance")
}
