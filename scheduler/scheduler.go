package scheduler

import (
	"sync"

	"github.com/docker/cli/cli/compose/types"
)

type Scheduler interface {
	SelectWithConstraints(resources *types.Resource) (string, error)
}

var schedulerInstance Scheduler
var once sync.Once

func GetScheduler() Scheduler {
	once.Do(func() {
		schedulerInstance = &NotSoSmartScheduler{}
	})
	return schedulerInstance
}
