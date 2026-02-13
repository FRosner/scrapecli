package main

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	prommodel "github.com/prometheus/common/model"
)

// special key used to count metrics that have no labels
const noneLabelKey = "<none>"

// parseScrape parses the Prometheus text exposition format from data and returns
// a sorted slice of MetricSummary containing name, type and description (help)
// and cardinality (number of metric instances / series).
// It also returns a map of global label values (label name -> set of values).
func parseScrape(data []byte) ([]MetricSummary, map[string]map[string]struct{}, error) {
	// Create a TextParser with explicit validation scheme to avoid relying on
	// global state. The zero value TextParser is invalid and may panic.
	parser := expfmt.NewTextParser(prommodel.UTF8Validation)
	mfs, err := parser.TextToMetricFamilies(bytes.NewReader(data))
	if err != nil {
		return nil, nil, err
	}

	// Global map to track distinct values for each label across all metrics
	globalValues := make(map[string]map[string]struct{})

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

		// Collect unique label names appearing in this metric family
		labelSet := make(map[string]struct{})
		for _, m := range mf.Metric {
			for _, lp := range m.Label {
				if lp.Name != nil {
					ln := *lp.Name
					labelSet[ln] = struct{}{}

					// Track global label values
					if _, ok := globalValues[ln]; !ok {
						globalValues[ln] = make(map[string]struct{})
					}
					if lp.Value != nil {
						globalValues[ln][*lp.Value] = struct{}{}
					}
				}
			}
		}

		// For histograms and summaries, a single Metric instance may contain
		// multiple exposed series (buckets/quantiles + sum/count). Compute a
		// more accurate series count for these types.
		switch mf.GetType() {
		case dto.MetricType_HISTOGRAM:
			// Sum buckets across all Metric entries
			card = 0
			if _, ok := globalValues["le"]; !ok {
				globalValues["le"] = make(map[string]struct{})
			}
			for _, metric := range mf.Metric {
				if metric.GetHistogram() != nil {
					buckets := metric.GetHistogram().Bucket
					card += len(buckets)
					for _, b := range buckets {
						if b.UpperBound != nil {
							val := fmt.Sprintf("%g", *b.UpperBound)
							globalValues["le"][val] = struct{}{}
						}
					}
				}
			}
			// Histograms implicitly have the "le" label on buckets
			if card > 0 {
				labelSet["le"] = struct{}{}
			}
		case dto.MetricType_SUMMARY:
			// Sum quantiles across all Metric entries
			card = 0
			if _, ok := globalValues["quantile"]; !ok {
				globalValues["quantile"] = make(map[string]struct{})
			}
			for _, metric := range mf.Metric {
				if metric.GetSummary() != nil {
					quantiles := metric.GetSummary().Quantile
					card += len(quantiles)
					for _, q := range quantiles {
						if q.Quantile != nil {
							val := fmt.Sprintf("%g", *q.Quantile)
							globalValues["quantile"][val] = struct{}{}
						}
					}
				}
			}
			// Summaries implicitly have the "quantile" label on quantiles
			if card > 0 {
				labelSet["quantile"] = struct{}{}
			}
		}

		metricLabels := make([]string, 0, len(labelSet))
		for l := range labelSet {
			metricLabels = append(metricLabels, l)
		}
		sort.Strings(metricLabels)

		m := MetricSummary{
			Name:        mf.GetName(),
			Type:        mf.GetType().String(),
			Description: mf.GetHelp(),
			Cardinality: card,
			Labels:      metricLabels,
			Size:        0, // filled later by scanning text lines
		}
		metrics = append(metrics, m)
	}

	// Compute size per metric by scanning the raw text representation line by line.
	// We'll count the bytes of any line that contains a metric name.
	// Build a lookup map from metric name to index in metrics slice for fast updates.
	nameToIdx := make(map[string]int, len(metrics))
	for i, m := range metrics {
		nameToIdx[m.Name] = i
	}

	// Split data into lines preserving newline lengths. We treat '\n' as line
	// separator and add 1 byte for it if present in original data.
	lines := bytes.Split(data, []byte("\n"))
	for i, line := range lines {
		lineLen := len(line)
		// Determine if this line had a trailing newline in the original data.
		// All but the last split part had a trailing '\n'.
		if i < len(lines)-1 {
			lineLen += 1 // include the '\n' byte
		}
		if lineLen == 0 {
			continue
		}
		// For each metric name, if it appears in the line, add the line length to that metric.
		// This is O(n*m) in worst case but metric lists are small for test files.
		lineStr := string(line)
		for name, idx := range nameToIdx {
			if strings.Contains(lineStr, name) {
				metrics[idx].Size += int64(lineLen)
			}
		}
	}

	return metrics, globalValues, nil
}

// SummarizeScrape composes all available summaries for a scrape.
func SummarizeScrape(data []byte) ScrapeSummary {
	metrics, globalValues, err := parseScrape(data)
	var top []CardinalityEntry
	if err != nil {
		// If parsing fails, return size summary and an empty metrics slice.
		// We avoid exiting here so callers can handle the summary as needed.
		metrics = []MetricSummary{}
		globalValues = make(map[string]map[string]struct{})
	} else {
		// Compute top 10 metrics by cardinality
		sort.Slice(metrics, func(i, j int) bool {
			return metrics[i].Cardinality > metrics[j].Cardinality
		})
		limit := 10
		if len(metrics) < limit {
			limit = len(metrics)
		}
		for i := 0; i < limit; i++ {
			top = append(top, CardinalityEntry{
				Name:        metrics[i].Name,
				Cardinality: metrics[i].Cardinality,
			})
		}
	}

	// Compute counts of all metric types (lowercased) so they can be included
	// in the summary.
	typesCount := make(map[string]int)
	labelCounts := make(map[string]int)
	labelValueCounts := make(map[string]int)

	// Ensure the none key exists so callers always see it even if zero
	labelCounts[noneLabelKey] = 0

	// Aggregate global label value counts
	for l, values := range globalValues {
		labelValueCounts[l] = len(values)
	}

	for _, m := range metrics {
		t := strings.ToLower(m.Type)
		typesCount[t]++

		// If a metric has no labels, count it under the special noneLabelKey
		if len(m.Labels) == 0 {
			labelCounts[noneLabelKey]++
			continue
		}

		for _, l := range m.Labels {
			labelCounts[l]++
		}
	}

	return ScrapeSummary{
		Summary: MetricsSummary{
			Bytes:            SummarizeSize(data).Bytes,
			TopCardinalities: top,
			TypesCount:       typesCount,
			LabelCounts:      labelCounts,
			LabelValueCounts: labelValueCounts,
		},
		Metrics: metrics,
	}
}
