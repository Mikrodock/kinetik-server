package models

import (
	composeTypes "github.com/docker/cli/cli/compose/types"
	"github.com/docker/docker/api/types"
)

type Service struct {
	StackName       string
	ServiceName     string
	ContainerConfig *types.ContainerCreateConfig
	Instances       []*Instance
	Constraints     *composeTypes.Resource
	Ports           []composeTypes.ServicePortConfig
}

func NewService(stackName, serviceName string, cntConfig *types.ContainerCreateConfig) *Service {
	return &Service{
		StackName:       stackName,
		ServiceName:     serviceName,
		ContainerConfig: cntConfig,
		Instances:       make([]*Instance, 0),
	}
}

func (s *Service) AddInstance(instance *Instance) {
	s.Instances = append(s.Instances, instance)
}

func (s *Service) GetInstances() []*Instance {
	return s.Instances
}
