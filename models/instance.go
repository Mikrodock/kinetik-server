package models

import "time"

type Instance struct {
	ID                *int                `json:"id,omitempty"`
	ServiceID         *int                `json:"service_id,omitempty"`
	NodeID            *int                `json:"node_id,omitempty"`
	CreationDate      time.Time           `json:"creation_date,omitempty"`
	UpdateDate        time.Time           `json:"update_date,omitempty"`
	State             *StateValue         `json:"state,omitempty"`
	RegisteredMetrics []*MetricDescriptor `json:"registered_metrics,omitempty"`
	Metrics           []*MetricValue      `json:"metrics,omitempty"`
	Timeout           *int                `json:"timeout,omitempty"`
}
