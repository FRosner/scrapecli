package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/fatih/color"
)

// FormatScrapeSummaryTerminal returns a human-readable, colored terminal
// representation of a ScrapeSummary. It relies on fatih/color for bold/colored
// text.
func FormatScrapeSummaryTerminal(s ScrapeSummary) string {
	var b strings.Builder

	bold := color.New(color.Bold).SprintFunc()
	cyan := color.New(color.FgHiCyan).SprintFunc()
	yellow := color.New(color.FgHiYellow).SprintFunc()
	green := color.New(color.FgHiGreen).SprintFunc()
	// Use a faint/dim style for missing descriptions to show they are less
	// prominent.
	dim := color.New(color.Faint).SprintFunc()

	b.WriteString(bold("## Summary") + "\n\n")
	b.WriteString(fmt.Sprintf("Size: %s\n\n", cyan(humanReadableBytes(s.Summary.Bytes))))

	// Top metrics (previously "Top Cardinalities")
	if len(s.Summary.TopCardinalities) > 0 {
		b.WriteString("Top Metrics:\n")
		for i, e := range s.Summary.TopCardinalities {
			valueWord := "series"
			// Only the number is green; the word remains uncolored
			b.WriteString(fmt.Sprintf("  %2d. %s: %s %s\n", i+1, yellow(e.Name), green(fmt.Sprintf("%d", e.Cardinality)), valueWord))
		}
		b.WriteString("\n")
	}

	// Type counts (all metric types)
	if len(s.Summary.TypesCount) > 0 {
		// Convert map to slice for deterministic ordering: sort by count desc then name
		types := make([]struct {
			Name  string
			Count int
		}, 0, len(s.Summary.TypesCount))
		for k, v := range s.Summary.TypesCount {
			types = append(types, struct {
				Name  string
				Count int
			}{Name: k, Count: v})
		}
		sort.Slice(types, func(i, j int) bool {
			if types[i].Count == types[j].Count {
				return types[i].Name < types[j].Name
			}
			return types[i].Count > types[j].Count
		})

		b.WriteString("Types:\n")
		for _, t := range types {
			metricWord := "metrics"
			if t.Count == 1 {
				metricWord = "metric"
			}
			// Only the number is green
			b.WriteString(fmt.Sprintf("  - %s: %s %s\n", yellow(t.Name), green(fmt.Sprintf("%d", t.Count)), metricWord))
		}
		b.WriteString("\n")
	}

	// Label counts
	if len(s.Summary.LabelCounts) > 0 {
		// Convert map to slice for deterministic ordering: sort by count desc then name
		labels := make([]struct {
			Name  string
			Count int
		}, 0, len(s.Summary.LabelCounts))
		for k, v := range s.Summary.LabelCounts {
			labels = append(labels, struct {
				Name  string
				Count int
			}{Name: k, Count: v})
		}
		sort.Slice(labels, func(i, j int) bool {
			if labels[i].Count == labels[j].Count {
				return labels[i].Name < labels[j].Name
			}
			return labels[i].Count > labels[j].Count
		})

		b.WriteString("Labels:\n")
		for _, l := range labels {
			distinctValCount := s.Summary.LabelValueCounts[l.Name]

			metricWord := "metrics"
			if l.Count == 1 {
				metricWord = "metric"
			}

			valueWord := "values"
			if distinctValCount == 1 {
				valueWord = "value"
			}

			if l.Name == "<none>" {
				// Special handling for <none> key which won't have values
				// Only the number is green
				b.WriteString(fmt.Sprintf("  - %s: %s %s\n", yellow(l.Name), green(fmt.Sprintf("%d", l.Count)), metricWord))
			} else {
				// Only the numbers are green; the words remain uncolored
				b.WriteString(fmt.Sprintf("  - %s: %s, %s\n", yellow(l.Name), green(fmt.Sprintf("%d", l.Count))+" "+metricWord, green(fmt.Sprintf("%d", distinctValCount))+" "+valueWord))
			}
		}
		b.WriteString("\n")
	}

	// Metrics - render as simple blocks rather than a table
	b.WriteString(bold("## Metrics") + "\n\n")
	for _, m := range s.Metrics {
		name := yellow(m.Name)
		card := green(fmt.Sprintf("%d", m.Cardinality))
		mType := green(strings.ToLower(m.Type))

		labelsPart := ""
		if len(m.Labels) > 0 {
			coloredLabels := make([]string, len(m.Labels))
			for i, l := range m.Labels {
				coloredLabels[i] = green(l)
			}
			labelsPart = fmt.Sprintf(", labels: %s", strings.Join(coloredLabels, ", "))
		}

		// pluralize value/values for readability
		valueWord := "values"
		if m.Cardinality == 1 {
			valueWord = "value"
		}

		b.WriteString(fmt.Sprintf("%s (type %s, %s %s%s)\n", name, mType, card, valueWord, labelsPart))

		desc := m.Description
		if desc == "" {
			// Use a lightweight/dim color for missing descriptions.
			b.WriteString(fmt.Sprintf("%s\n\n", dim("<no description>")))
		} else {
			b.WriteString(fmt.Sprintf("%s\n\n", dim(desc)))
		}
	}

	return b.String()
}

// humanReadableBytes formats a byte count into a human-friendly string using
// binary units (KiB, MiB, ...). For values below 1024 it returns "<n> bytes".
func humanReadableBytes(b int64) string {
	if b < 1024 {
		return fmt.Sprintf("%d bytes", b)
	}
	units := []string{"KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	val := float64(b)
	i := 0
	for val >= 1024 && i < len(units) {
		val = val / 1024
		i++
	}
	// i is the number of divisions performed; since b >= 1024, i >= 1 and
	// the corresponding unit is units[i-1].
	unit := units[i-1]
	return fmt.Sprintf("%.2f %s", val, unit)
}
