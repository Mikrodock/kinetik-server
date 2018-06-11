package boltdb

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"kinetik-server/logger"
	"kinetik-server/models"
	"kinetik-server/models/internals"
	"os"
	"path"

	"github.com/boltdb/bolt"
)

const PATH = "/etc/kinetik"

func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func IsFirstRun() bool {
	if _, err := os.Stat(PATH); os.IsNotExist(err) {
		os.MkdirAll(PATH, 755)
		return true
	}
	return false
}

type BoltDB struct {
	client *bolt.DB
}

func NewBoltDB() (*BoltDB, error) {
	db, err := bolt.Open(path.Join(PATH, "kinetik.db"), 0600, nil)
	if err != nil {
		return nil, err
	}

	//Ensure buckets existance
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("nodes"))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists([]byte("instances"))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists([]byte("services"))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists([]byte("config"))
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return &BoltDB{
		client: db,
	}, nil
}

func (b *BoltDB) GetNodes() map[string]*models.Node {

	nodes := make(map[string]*models.Node, 0)

	b.client.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("nodes"))
		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var n models.Node
			if err := json.Unmarshal(v, &n); err != nil {
				return err
			}
			nodes[string(k)] = &n
		}

		return nil
	})

	return nodes
}

func (b *BoltDB) GetNode(nodeId string) *models.Node {

	var node *models.Node

	b.client.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("nodes"))
		nodeBytes := bucket.Get([]byte(nodeId))
		if nodeBytes != nil {
			return json.Unmarshal(nodeBytes, node)
		}
		return nil
	})

	return node
}

func (b *BoltDB) AddNode(nodeid string, node *models.Node) error {
	return b.client.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("nodes"))

		buf, err := json.Marshal(node)

		fmt.Printf("%s\n", string(buf))

		if err != nil {
			return err
		}

		return bucket.Put([]byte(nodeid), buf)
	})
}

func (b *BoltDB) DeleteNode(nodeID int) error {
	return b.client.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("nodes"))

		// TODO : delete all instances linked on this node and reschedule
		// That means also handle DO LB and so

		return bucket.Delete(itob(nodeID))
	})
}

func (b *BoltDB) GetServices() []*models.Service {
	services := make([]*models.Service, 0)

	b.client.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("services"))
		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var n models.Service
			if err := json.Unmarshal(v, &n); err != nil {
				return err
			}
			services = append(services, &n)
		}

		return nil
	})

	return services
}

func (b *BoltDB) GetService(identifier string) *models.Service {
	var service *models.Service = &models.Service{}

	b.client.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("services"))
		value := bucket.Get([]byte(identifier))
		logger.StdLog.Println(value)
		err := json.Unmarshal(value, service)
		if err != nil {
			logger.ErrLog.Println("Cannot unmarshal service", err)
		}
		return err
	})

	return service
}

func (b *BoltDB) AddService(service *models.Service) error {
	return b.client.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("services"))

		buf, err := json.Marshal(service)
		if err != nil {
			return err
		}

		return bucket.Put([]byte(service.StackName+"/"+service.ServiceName), buf)
	})
}

func (b *BoltDB) DeleteService(serviceID int) error {
	return b.client.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("services"))

		// TODO : Delete all instances linked to this service
		// That means also handle DO LB and so

		return bucket.Delete(itob(serviceID))
	})
}

func (b *BoltDB) GetInstances() []*models.Instance {
	instances := make([]*models.Instance, 0)

	b.client.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("instances"))
		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var n models.Instance
			if err := json.Unmarshal(v, &n); err != nil {
				return err
			}
			instances = append(instances, &n)
		}

		return nil
	})

	return instances
}

func (b *BoltDB) AddInstance(srv, stack string, instance *models.Instance) error {
	return b.client.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("instances"))

		subBucket, err := bucket.CreateBucketIfNotExists([]byte(stack + "/" + srv))

		if err != nil {
			return err
		}

		buf, err := json.Marshal(instance)
		if err != nil {
			return err
		}

		return subBucket.Put([]byte(instance.ContainerID), buf)
	})
}

func (b *BoltDB) DeleteInstance(instanceID int) error {
	return b.client.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("instances"))
		return bucket.Delete(itob(instanceID))
	})
}

func (b *BoltDB) SetConfig(config *internals.Config) error {
	bytes, err := json.Marshal(config)
	if err != nil {
		return err
	}
	return b.client.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("config"))
		return bucket.Put([]byte("config"), bytes)
	})
}

func (b *BoltDB) GetConfig() *internals.Config {

	var config internals.Config

	b.client.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("config"))

		b := bucket.Get([]byte("config"))

		json.Unmarshal(b, &config)

		return nil
	})

	return &config
}
