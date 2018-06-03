package scheduler

import (
	"kinetik-server/data"
	"math/rand"

	"github.com/docker/cli/cli/compose/types"
)

type DumbScheduler struct{}

func (ds *DumbScheduler) SelectWithConstraints(resources *types.Resource) (string, error) {
	nodes := data.GetDB().GetNodes()
	keys := make([]string, 0, len(nodes))
	for k := range nodes {
		keys = append(keys, k)
	}
	index := rand.Int() % len(keys)

	key := keys[index]

	return key, nil
}
