package models

type MetricBreakpoint struct {
	State *StateValue `json:"state,omitempty"`
	Value *float32    `json:"value,omitempty"`
}

type MetricDescriptor struct {
	Name        string              `json:"name,omitempty"`
	Breakpoints []*MetricBreakpoint `json:"breakpoints,omitempty"`
}

type MetricValue struct {
	Name  string   `json:"name,omitempty"`
	Value *float32 `json:"value,omitempty"`
}
