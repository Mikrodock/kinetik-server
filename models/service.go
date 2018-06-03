package models

import "time"

type Service struct {
	ID                *int                `json:"id,omitempty"`
	Name              string              `json:"name,omitempty"`
	CreationDate      time.Time           `json:"creation_date,omitempty"`
	UpdateDate        time.Time           `json:"update_date,omitempty"`
	GlobalState       *StateValue         `json:"global_state,omitempty"`
	RegisteredMetrics []*MetricDescriptor `json:"registered_metrics,omitempty"`
	Metrics           []*MetricValue      `json:"metrics,omitempty"`
	Instances         []*Instance         `json:"instances,omitempty"`
	Nodes             []*Node             `json:"nodes,omitempty"`
}
