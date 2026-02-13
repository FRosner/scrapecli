package main

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSummarizeScrape_Integration(t *testing.T) {
	f, err := os.Open("test-resources/prometheus-scrape.txt")
	require.NoError(t, err, "failed to open test resource")
	defer func() { _ = f.Close() }()

	data, err := io.ReadAll(f)
	require.NoError(t, err, "failed to read test resource")

	summary := SummarizeScrape(data)

	// Expected bytes taken from a previous run
	expected := int64(79033)
	require.Equal(t, expected, summary.Summary.Bytes, "unexpected byte count")

	// Metrics should be present
	require.Greater(t, len(summary.Metrics), 0, "expected at least one metric")

	// Build map for easy lookups
	metricsByName := make(map[string]MetricSummary, len(summary.Metrics))
	for _, m := range summary.Metrics {
		metricsByName[m.Name] = m
	}

	// Assert presence and metadata for known metrics and their cardinalities
	g, ok := metricsByName["go_goroutines"]
	require.True(t, ok, "go_goroutines metric not found in parsed metrics")
	require.Equal(t, "GAUGE", g.Type, "unexpected type for go_goroutines")
	require.Contains(t, g.Description, "Number of goroutines", "unexpected description for go_goroutines")
	require.Equal(t, 1, g.Cardinality, "unexpected cardinality for go_goroutines")

	b, ok := metricsByName["go_gc_heap_allocs_by_size_bytes"]
	require.True(t, ok, "go_gc_heap_allocs_by_size_bytes metric not found in parsed metrics")
	// For the histogram family, each bucket/sample is represented by a Metric instance
	require.Equal(t, 12, b.Cardinality, "unexpected cardinality for go_gc_heap_allocs_by_size_bytes")
	require.Equal(t, "HISTOGRAM", b.Type, "unexpected type for go_gc_heap_allocs_by_size_bytes")
	require.Contains(t, b.Description, "Distribution of heap allocations by approximate size", "unexpected description for go_gc_heap_allocs_by_size_bytes")

	c, ok := metricsByName["prometheus_tsdb_exemplar_exemplars_appended_total"]
	require.True(t, ok, "prometheus_tsdb_exemplar_exemplars_appended_total metric not found in parsed metrics")
	require.Equal(t, "COUNTER", c.Type, "unexpected type for prometheus_tsdb_exemplar_exemplars_appended_total")
	require.Contains(t, c.Description, "Total number of appended exemplars", "unexpected description for prometheus_tsdb_exemplar_exemplars_appended_total")
	require.Equal(t, 1, c.Cardinality, "unexpected cardinality for prometheus_tsdb_exemplar_exemplars_appended_total")

	// Validate top cardinalities summary
	top := summary.Summary.TopCardinalities
	require.Greater(t, len(top), 0, "expected at least one top cardinality entry")
	require.LessOrEqual(t, len(top), 10, "top cardinalities should contain at most 10 entries")

	// Top list should be sorted in non-increasing order of cardinality
	for i := 1; i < len(top); i++ {
		require.GreaterOrEqual(t, top[i-1].Cardinality, top[i].Cardinality, "top cardinalities not sorted descending")
	}

	// Each top entry should correspond to a metric and have matching cardinality
	foundBucketInTop := false
	for _, entry := range top {
		m, ok := metricsByName[entry.Name]
		require.True(t, ok, "top cardinality metric %s not present in metrics list", entry.Name)
		require.Equal(t, m.Cardinality, entry.Cardinality, "cardinality mismatch for top metric %s", entry.Name)
		if entry.Name == "go_gc_heap_allocs_by_size_bytes" {
			foundBucketInTop = true
		}
	}
	// Ensure a known high-cardinality metric is present in the top list
	require.True(t, foundBucketInTop, "expected go_gc_heap_allocs_by_size_bytes to be present in top cardinalities")

	// New assertions: verify the TypesCount in the summary matches counts computed from the parsed metrics.
	typesFromMetrics := make(map[string]int)
	for _, m := range summary.Metrics {
		typesFromMetrics[strings.ToLower(m.Type)]++
	}

	require.Equal(t, typesFromMetrics, summary.Summary.TypesCount, "type counts in summary should match counts from parsed metrics")

	// Verify labels for select metrics.
	// go_gc_heap_allocs_by_size_bytes is a histogram, so it must have "le" label.
	// It is fetched into 'b' above.
	require.Contains(t, b.Labels, "le", "go_gc_heap_allocs_by_size_bytes should have 'le' label")

	// Verify LabelCounts consistency
	labelCountsFromMetrics := make(map[string]int)
	// Include the special none key so expected matches summary behavior for metrics without labels
	labelCountsFromMetrics[noneLabelKey] = 0
	for _, m := range summary.Metrics {
		if len(m.Labels) == 0 {
			labelCountsFromMetrics[noneLabelKey]++
			continue
		}
		for _, l := range m.Labels {
			labelCountsFromMetrics[l]++
		}
	}
	require.Equal(t, labelCountsFromMetrics, summary.Summary.LabelCounts, "label counts in summary should match counts from parsed metrics")
}
