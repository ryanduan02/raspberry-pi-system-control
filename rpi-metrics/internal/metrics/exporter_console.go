package metrics

import (
	"encoding/json"
	"io"
	"time"
)

type ConsoleExporter struct {
	Out io.Writer
}

type consoleEnvelope struct {
	CollectedAt time.Time         `json:"collected_at"`
	Samples     []Sample          `json:"samples"`
	Errors      []CollectorError  `json:"errors,omitempty"`
}

func (e ConsoleExporter) Export(res Result) error {
	env := consoleEnvelope{
		CollectedAt: time.Now().UTC(),
		Samples:     res.Samples,
		Errors:      res.Errors,
	}

	enc := json.NewEncoder(e.Out)
	enc.SetEscapeHTML(false)
	return enc.Encode(env) // JSON Lines: one object per interval
}
