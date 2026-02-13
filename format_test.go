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
			{Name: "metric_high_card", Type: "GAUGE", Description: "A high cardinality metric", Cardinality: 100, Labels: []string{"env"}},
			{Name: "metric_low_card", Type: "COUNTER", Description: "A low cardinality metric", Cardinality: 2, Labels: []string{"env", "job"}},
			{Name: "metric_no_desc", Type: "GAUGE", Description: "", Cardinality: 1},
		},
	}

	out := FormatScrapeSummaryTerminal(s)

	expected := `## Summary

Size: 12.06 KiB

Top Cardinalities:
   1. metric_high_card: 100
   2. metric_low_card: 2

Types:
  - gauge: 2 metrics
  - counter: 1 metric

Labels:
  - env: 2 distinct metrics, 3 distinct values
  - job: 1 distinct metric, 1 distinct value

## Metrics

metric_high_card (type gauge, cardinality 100, labels: env)
A high cardinality metric

metric_low_card (type counter, cardinality 2, labels: env, job)
A low cardinality metric

metric_no_desc (type gauge, cardinality 1)
<no description>

`

	require.Equal(t, expected, out, "formatted output should match exactly")
}
