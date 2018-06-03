package models

import (
	"github.com/docker/cli/cli/compose/types"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/load"
)

type Node struct {
	*load.AvgStat
	MemUsedPercent float64         `json:"mem_used_percent,omitempty"`
	MemUsedBytes   uint64          `json:"mem_used_bytes,omitempty"`
	CPUUsedPercent float64         `json:"cpu_used_percent,omitempty"`
	CPUCount       int             `json:"cpu_count,omitempty"`
	DiskUsage      *disk.UsageStat `json:"disk_usage,omitempty"`

	Reservations *types.Resource
}
