package metrics

import "context"

type Exporter interface {
	Export(ctx context.Context, res Result) error
}
