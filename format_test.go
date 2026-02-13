package main

import (
	"testing"

	"github.com/fatih/color"
	"github.com/stretchr/testify/require"
)

func TestFormatScrapeSummaryTerminal(t *testing.T) {
	// Ensure deterministic output (no ANSI color codes)
	color.NoColor = true

	// Build a dummy ScrapeSummary that exercises every field
	s := ScrapeSummary{
		Summary: MetricsSummary{
			Bytes: 12345,
			TopCardinalities: []CardinalityEntry{
				{Name: "metric_high_card", Cardinality: 100},
				{Name: "metric_low_card", Cardinality: 2},
			},
			TypesCount: map[string]int{"gauge": 2, "counter": 1},
			LabelCounts: map[string]int{
				"env": 2,
				"job": 1,
			},
			LabelValueCounts: map[string]int{
				"env": 3,
				"job": 1,
			},
		},
		Metrics: []MetricSummary{
			{Name: "metric_high_card", Type: "GAUGE", Description: "A high cardinality metric", Cardinality: 100, Labels: []string{"env"}, Size: 10240},
			{Name: "metric_low_card", Type: "COUNTER", Description: "A low cardinality metric", Cardinality: 2, Labels: []string{"env", "job"}, Size: 1024},
			{Name: "metric_no_desc", Type: "GAUGE", Description: "", Cardinality: 1},
		},
	}

	out := FormatScrapeSummaryTerminal(s)

	expected := `## Summary

Size: 12.06 KiB

Top Metrics:
   1. metric_high_card: 100 series, 10.00 KiB
   2. metric_low_card: 2 series, 1.00 KiB

Types:
  - gauge: 2 metrics
  - counter: 1 metric

Labels:
  - env: 3 values, 2 metrics
  - job: 1 value, 1 metric

## Metrics

metric_high_card (type gauge, 100 values, labels: env)
A high cardinality metric

metric_low_card (type counter, 2 values, labels: env, job)
A low cardinality metric

metric_no_desc (type gauge, 1 value)
<no description>

`

	require.Equal(t, expected, out, "formatted output should match exactly")
}
