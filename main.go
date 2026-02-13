package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	prommodel "github.com/prometheus/common/model"
)

// MetricsSummary holds a summary of the size and is JSON-serializable.
type MetricsSummary struct {
	Bytes            int64              `json:"bytes"`
	TopCardinalities []CardinalityEntry `json:"top_cardinalities"`
}

// CardinalityEntry is a small struct holding metric name and its cardinality.
type CardinalityEntry struct {
	Name        string `json:"name"`
	Cardinality int    `json:"cardinality"`
}

// MetricSummary holds minimal metadata about a metric.
type MetricSummary struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Cardinality int    `json:"cardinality"`
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
// a sorted slice of MetricSummary containing name, type and description (help)
// and cardinality (number of metric instances / series).
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

		// Default cardinality is number of Metric instances in the family
		card := len(mf.Metric)

		// For histograms and summaries, a single Metric instance may contain
		// multiple exposed series (buckets/quantiles + sum/count). Compute a
		// more accurate series count for these types.
		switch mf.GetType() {
		case dto.MetricType_HISTOGRAM:
			// Sum buckets across all Metric entries
			card = 0
			for _, metric := range mf.Metric {
				if metric.GetHistogram() != nil {
					card += len(metric.GetHistogram().Bucket)
				}
			}
		case dto.MetricType_SUMMARY:
			// Sum quantiles across all Metric entries
			card = 0
			for _, metric := range mf.Metric {
				if metric.GetSummary() != nil {
					card += len(metric.GetSummary().Quantile)
				}
			}
		}

		m := MetricSummary{
			Name:        mf.GetName(),
			Type:        mf.GetType().String(),
			Description: mf.GetHelp(),
			Cardinality: card,
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}

// SummarizeScrape composes all available summaries for a scrape.
func SummarizeScrape(data []byte) ScrapeSummary {
	metrics, err := parseScrape(data)
	var top []CardinalityEntry
	if err != nil {
		// If parsing fails, return size summary and an empty metrics slice.
		// We avoid exiting here so callers can handle the summary as needed.
		metrics = []MetricSummary{}
	} else {
		// Compute top 10 metrics by cardinality
		sort.Slice(metrics, func(i, j int) bool {
			return metrics[i].Cardinality > metrics[j].Cardinality
		})
		max := 10
		if len(metrics) < max {
			max = len(metrics)
		}
		for i := 0; i < max; i++ {
			top = append(top, CardinalityEntry{
				Name:        metrics[i].Name,
				Cardinality: metrics[i].Cardinality,
			})
		}
	}

	return ScrapeSummary{
		Summary: MetricsSummary{
			Bytes:            SummarizeSize(data).Bytes,
			TopCardinalities: top,
		},
		Metrics: metrics,
	}
}

func main() {
	// Read entire Prometheus scrape from stdin
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		_ = fmt.Errorf("error reading stdin: %v\n", err)
		os.Exit(1)
	}

	summary := SummarizeScrape(data)

	b, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		_ = fmt.Errorf("error marshaling json: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(b))
}
