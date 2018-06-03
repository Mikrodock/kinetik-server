package scheduler

import (
	"kinetik-server/data"
	"kinetik-server/logger"
	"kinetik-server/models"
	"strconv"

	"github.com/docker/cli/cli/compose/types"
)

type NotSoSmartScheduler struct{}

func (ds *NotSoSmartScheduler) SelectWithConstraints(resources *types.Resource) (string, error) {

	var node *models.Node
	var ipnode string

	nodes := data.GetDB().GetNodes()
	for ip, nodeSpec := range nodes {
		if resources == nil {
			return ip, nil
		}
		nodeCpuReservation, _ := strconv.ParseFloat(nodeSpec.Reservations.NanoCPUs, 64)
		thisSpecCpuReservation, _ := strconv.ParseFloat(resources.NanoCPUs, 64)
		freeCPUpercents := float64(100*nodeSpec.CPUCount) - nodeSpec.CPUUsedPercent
		realFree := freeCPUpercents - nodeCpuReservation*100
		if thisSpecCpuReservation*100 < realFree {
			memBytesFree := int64((float64(nodeSpec.MemUsedBytes) / nodeSpec.MemUsedPercent) * (1 - nodeSpec.MemUsedPercent))
			if int64(resources.MemoryBytes) < memBytesFree-int64(nodeSpec.Reservations.MemoryBytes) {
				nodeSpec.Reservations.NanoCPUs = strconv.FormatFloat(nodeCpuReservation+thisSpecCpuReservation, 'f', -1, 64)
				nodeSpec.Reservations.MemoryBytes = nodeSpec.Reservations.MemoryBytes + resources.MemoryBytes
				node = nodeSpec
				ipnode = ip
				break
			}
		}
	}

	if node != nil {
		data.GetDB().AddNode(ipnode, node)

		logger.StdLog.Printf("Node %s has now CPU res = %s and mem = %d\n", ipnode, node.Reservations.NanoCPUs, node.Reservations.MemoryBytes)

		return ipnode, nil
	}

	return "", nil
}
