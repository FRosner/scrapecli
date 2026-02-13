package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/prometheus/common/expfmt"
	prommodel "github.com/prometheus/common/model"
)

// MetricsSummary holds a summary of the size and is JSON-serializable.
type MetricsSummary struct {
	Bytes int64 `json:"bytes"`
}

// MetricSummary holds minimal metadata about a metric.
type MetricSummary struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// ScrapeSummary wraps different summaries about a scrape.
type ScrapeSummary struct {
	Summary MetricsSummary  `json:"summary"`
	Metrics []MetricSummary `json:"metrics"`
}

// SummarizeSize takes the raw scrape bytes and returns a MetricsSummary.
func SummarizeSize(data []byte) MetricsSummary {
	return MetricsSummary{
		Bytes: int64(len(data)),
	}
}

// parseScrape parses the Prometheus text exposition format from data and returns
// a sorted slice of MetricSummary containing name, type and description (help).
func parseScrape(data []byte) ([]MetricSummary, error) {
	// Create a TextParser with explicit validation scheme to avoid relying on
	// global state. The zero value TextParser is invalid and may panic.
	parser := expfmt.NewTextParser(prommodel.UTF8Validation)
	mfs, err := parser.TextToMetricFamilies(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	// Collect and sort metric names for deterministic output
	names := make([]string, 0, len(mfs))
	for name := range mfs {
		names = append(names, name)
	}
	sort.Strings(names)

	metrics := make([]MetricSummary, 0, len(names))
	for _, name := range names {
		mf := mfs[name]
		m := MetricSummary{
			Name:        mf.GetName(),
			Type:        mf.GetType().String(),
			Description: mf.GetHelp(),
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}

// SummarizeScrape composes all available summaries for a scrape.
func SummarizeScrape(data []byte) ScrapeSummary {
	metrics, err := parseScrape(data)
	if err != nil {
		// If parsing fails, return size summary and an empty metrics slice.
		// We avoid exiting here so callers can handle the summary as needed.
		metrics = []MetricSummary{}
	}

	return ScrapeSummary{
		Summary: SummarizeSize(data),
		Metrics: metrics,
	}
}

func main() {
	// Read entire Prometheus scrape from stdin
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading stdin: %v\n", err)
		os.Exit(1)
	}

	summary := SummarizeScrape(data)

	b, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshaling json: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(b))
}
