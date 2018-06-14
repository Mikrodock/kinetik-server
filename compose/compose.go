package compose

import (
	"os"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"

	"github.com/docker/cli/cli/compose/loader"
	"github.com/docker/cli/cli/compose/types"
)

func LoadYAMLWithEnv(yaml []byte, env map[string]string) (*types.Config, error) {
	dict, err := loader.ParseYAML(yaml)
	if err != nil {
		return nil, err
	}
	workingDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	return loader.Load(types.ConfigDetails{
		WorkingDir: workingDir,
		ConfigFiles: []types.ConfigFile{
			{Filename: "docker-compose.yml", Config: dict},
		},
		Environment: nil,
	})
}

func convertEnv(srvEnv types.MappingWithEquals) []string {
	envs := make([]string, 0)
	for k, v := range srvEnv {
		if v != nil {
			envs = append(envs, k+"="+*v)
		}
	}
	return envs
}

func ConvertServiceToContainer(srvConfig *types.ServiceConfig) (*dockerTypes.ContainerCreateConfig, error) {
	cntCreateConfig := &dockerTypes.ContainerCreateConfig{
		Name: srvConfig.ContainerName,
		Config: &container.Config{
			Cmd:         []string(srvConfig.Command),
			Domainname:  srvConfig.DomainName,
			Entrypoint:  []string(srvConfig.Entrypoint),
			Env:         convertEnv(srvConfig.Environment),
			Healthcheck: convertHC(srvConfig.HealthCheck),
			Hostname:    srvConfig.Hostname,
			Image:       srvConfig.Image,
			// Labels also
			MacAddress: srvConfig.MacAddress,
			OpenStdin:  srvConfig.StdinOpen,
			StopSignal: srvConfig.StopSignal,
			Tty:        srvConfig.Tty,
			User:       srvConfig.User,
			WorkingDir: srvConfig.WorkingDir,
		},
		HostConfig: &container.HostConfig{
			CapAdd:     srvConfig.CapAdd,
			CapDrop:    srvConfig.CapDrop,
			ExtraHosts: srvConfig.ExtraHosts,
			// DNS => use MikroDNS
			// DNSSearch => stack.mikrodock
			Privileged: srvConfig.Privileged,
			RestartPolicy: container.RestartPolicy{
				Name:              srvConfig.Restart,
				MaximumRetryCount: 10,
			},
		},
		NetworkingConfig: &network.NetworkingConfig{},
	}

	return cntCreateConfig, nil
}

func convertHC(composehealth *types.HealthCheckConfig) *container.HealthConfig {
	ret := &container.HealthConfig{
		Test: composehealth.Test,
	}
	if composehealth.Interval != nil {
		ret.Interval = *composehealth.Interval
	}
	if composehealth.Retries != nil {
		ret.Retries = int(*composehealth.Retries)
	}
	if composehealth.StartPeriod != nil {
		ret.StartPeriod = *composehealth.StartPeriod
	}
	if composehealth.Timeout != nil {
		ret.Timeout = *composehealth.Timeout
	}
	return ret
}
