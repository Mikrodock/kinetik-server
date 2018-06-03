package data

import (
	"kinetik-server/boltdb"
	"kinetik-server/models"
	"kinetik-server/models/internals"
	"sync"
)

type DataHandler interface {
	GetNodes() map[string]*models.Node
	GetNode(nodeId string) *models.Node
	AddNode(nodeid string, node *models.Node) error
	DeleteNode(nodeID int) error
	GetServices() []*models.Service
	AddService(service *models.Service) error
	DeleteService(serviceID int) error
	GetInstances() []*models.Instance
	AddInstance(instance *models.Instance) error
	DeleteInstance(instanceID int) error
	SetConfig(config *internals.Config) error
	GetConfig() *internals.Config
}

var dbInstance DataHandler
var once sync.Once

func GetDB() DataHandler {
	once.Do(func() {
		dbInstance, _ = boltdb.NewBoltDB()
	})
	return dbInstance
}
