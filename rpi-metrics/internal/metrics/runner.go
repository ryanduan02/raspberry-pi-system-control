package metrics

import (
	"context"
	"time"
)

type Runner struct {
	Collectors []Collector
}

type Result struct {
	Samples []Sample
	Errors  []CollectorError
}

type CollectorError struct {
	CollectorID string `json:"collector"`
	Error       string `json:"error"`
}

func (r Runner) CollectOnce(ctx context.Context) Result {
	var res Result

	now := time.Now().UTC()
	for _, c := range r.Collectors {
		samples, err := c.Collect(ctx)
		if err != nil {
			res.Errors = append(res.Errors, CollectorError{
				CollectorID: c.ID(),
				Error:       err.Error(),
			})
			continue
		}
		// Ensure timestamp is present; collectors may also set their own.
		for i := range samples {
			if samples[i].Timestamp.IsZero() {
				samples[i].Timestamp = now
			}
		}
		res.Samples = append(res.Samples, samples...)
	}

	return res
}
