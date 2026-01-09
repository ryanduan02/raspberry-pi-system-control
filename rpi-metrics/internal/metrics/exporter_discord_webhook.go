package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"rpi-metrics/constants"
)

type DiscordWebhookExporter struct {
	WebhookURL string
	Client     *http.Client

	// If > 0, limits how often we post (prevents spam/rate-limit issues)
	MinInterval time.Duration
	lastSent    time.Time
}

type discordWebhookPayload struct {
	Content string `json:"content"`
}

func (e *DiscordWebhookExporter) Export(ctx context.Context, res Result) error {
	if e.WebhookURL == "" {
		return fmt.Errorf("discord webhook url is empty")
	}

	if e.Client == nil {
		e.Client = &http.Client{Timeout: 10 * time.Second}
	}

	now := time.Now()
	if e.MinInterval > 0 && !e.lastSent.IsZero() && now.Sub(e.lastSent) < e.MinInterval {
		return nil // too soon; skip
	}

	msg := formatDiscordMessage(res)

	bodyBytes, err := json.Marshal(discordWebhookPayload{Content: msg})
	if err != nil {
		return fmt.Errorf("marshal discord payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.WebhookURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.Client.Do(req)
	if err != nil {
		return fmt.Errorf("post to discord webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		if len(b) > 0 {
			return fmt.Errorf("discord webhook returned status %d: %s", resp.StatusCode, string(b))
		}
		return fmt.Errorf("discord webhook returned status %d", resp.StatusCode)
	}

	e.lastSent = now
	return nil
}

func formatDiscordMessage(res Result) string {
	collectedAt := time.Now().UTC()
	for _, s := range res.Samples {
		if !s.Timestamp.IsZero() {
			collectedAt = s.Timestamp.UTC()
			break
		}
	}

	separator := strings.Repeat("-", constants.DiscordMessageSeparatorLen)
	lines := fmt.Sprintf("%s\nMetrics (collected at %s):", separator, collectedAt.Format(time.RFC3339))

	// Print one line per metric sample.
	for _, s := range res.Samples {
		unit := s.Unit
		if unit == "" {
			unit = "(no unit)"
		}

		if unit == "bytes" {
			lines += fmt.Sprintf("\n%s: %s", s.Name, fmtBytesAsGigabytesWithRawBytes(s.Value))
			continue
		}
		if unit == "celsius" {
			lines += fmt.Sprintf("\n%s: %.3f celsius", s.Name, s.Value)
			continue
		}
		lines += fmt.Sprintf("\n%s: %.3f %s", s.Name, s.Value, unit)
	}

	if len(res.Errors) > 0 {
		lines += "\nErrors:"
		for _, e := range res.Errors {
			lines += fmt.Sprintf("\n- %s: %s", e.CollectorID, e.Error)
		}
	}
	return lines
}

func fmtBytesAsGigabytesWithRawBytes(v float64) string {
	// Uses decimal gigabytes (1 GB = 1,000,000,000 bytes).
	if v < 0 {
		v = 0
	}
	gb := v / 1_000_000_000.0
	return fmt.Sprintf("%.3f gigabytes (%.3f bytes)", gb, v)
}
