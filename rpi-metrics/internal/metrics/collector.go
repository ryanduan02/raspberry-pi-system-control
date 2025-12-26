package metrics

import "context"

type Collector interface {
	ID() string
	Collect(ctx context.Context) ([]Sample, error)
}
