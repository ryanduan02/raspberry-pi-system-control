package metrics

import "time"

type Sample struct {
	Name      string            `json:"name"`
	Value     float64           `json:"value"`
	Unit      string            `json:"unit,omitempty"`
	Timestamp time.Time         `json:"ts"`
	Labels    map[string]string `json:"labels,omitempty"`
}
