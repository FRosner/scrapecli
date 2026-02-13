package main

import (
	"bytes"
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

		// Collect unique label names appearing in this metric family
		labelSet := make(map[string]struct{})
		for _, m := range mf.Metric {
			for _, lp := range m.Label {
				if lp.Name != nil {
					labelSet[*lp.Name] = struct{}{}
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
			for _, metric := range mf.Metric {
				if metric.GetHistogram() != nil {
					card += len(metric.GetHistogram().Bucket)
				}
			}
			// Histograms implicitly have the "le" label on buckets
			if card > 0 {
				labelSet["le"] = struct{}{}
			}
		case dto.MetricType_SUMMARY:
			// Sum quantiles across all Metric entries
			card = 0
			for _, metric := range mf.Metric {
				if metric.GetSummary() != nil {
					card += len(metric.GetSummary().Quantile)
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

	// Compute counts of all metric types (lowercased) so they can be included
	// in the summary.
	typesCount := make(map[string]int)
	labelCounts := make(map[string]int)

	// Ensure the none key exists so callers always see it even if zero
	labelCounts[noneLabelKey] = 0

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
		},
		Metrics: metrics,
	}
}
