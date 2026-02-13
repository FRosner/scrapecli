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
		},
		Metrics: []MetricSummary{
			{Name: "metric_high_card", Type: "GAUGE", Description: "A high cardinality metric", Cardinality: 100},
			{Name: "metric_low_card", Type: "COUNTER", Description: "A low cardinality metric", Cardinality: 2},
			{Name: "metric_no_desc", Type: "GAUGE", Description: "", Cardinality: 1},
		},
	}

	out := FormatScrapeSummaryTerminal(s)

	expected := `Scrape Summary

Size: 12.06 KiB

Top Cardinalities:
   1. metric_high_card: 100
   2. metric_low_card: 2

Metrics

metric_high_card (type gauge, cardinality 100)
A high cardinality metric

metric_low_card (type counter, cardinality 2)
A low cardinality metric

metric_no_desc (type gauge, cardinality 1)
no description

`

	require.Equal(t, expected, out, "formatted output should match exactly")
}
