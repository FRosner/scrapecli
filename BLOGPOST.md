---
title: Taming Prometheus Scrapes - Understanding and Analyzing Your Metrics Endpoints
published: true
description: Learn how to diagnose oversized Prometheus scrapes and high-cardinality metrics using shell tools and scrapecli, a small CLI that parses the exposition format and gives you accurate series counts, type breakdowns, and label analysis.
tags: golang, prometheus, devops, observability
cover_image: https://dev-to-uploads.s3.amazonaws.com/uploads/articles/hgb66glgt7ifn5bl4ppu.png
---

## Introduction: Prometheus and the Scrape Format

Prometheus is a pull-based monitoring system. Instead of having services push metrics to a central collector, Prometheus periodically fetches (or *scrapes*) an HTTP endpoint (typically `/metrics`) from each monitored target. The response is a plain-text document in the [Prometheus exposition format](https://prometheus.io/docs/instrumenting/exposition_formats/), listing every metric the service wants to expose. A scrape looks something like this:

```
# HELP http_requests_total Total number of HTTP requests.
# TYPE http_requests_total counter
http_requests_total{method="GET",status="200"} 1234
http_requests_total{method="GET",status="500"} 7
http_requests_total{method="POST",status="200"} 89

# HELP process_resident_memory_bytes Resident memory size in bytes.
# TYPE process_resident_memory_bytes gauge
process_resident_memory_bytes 52428800
```

Each metric family starts with optional `# HELP` (description) and `# TYPE` (counter, gauge, histogram, summary, or untyped) comment lines, followed by one or more *series*: individual data points distinguished by their *label sets*. In the example above, `http_requests_total` has three series, one for each combination of `method` and `status` labels.

### Metric Types

Prometheus defines four main metric types:

- **Counter**: A monotonically increasing value (e.g. total requests, total errors).
- **Gauge**: A value that can go up or down (e.g. current memory usage, queue depth).
- **Histogram**: Samples observations into configurable buckets, plus a `_sum` and `_count`. Used for measuring things like request latencies or payload sizes. Each bucket boundary becomes its own series, labeled with `le` ("less than or equal").
- **Summary**: Similar to a histogram but pre-computes quantiles client-side, labeled with `quantile`.

Histograms and summaries deserve a special mention when thinking about cardinality. A single histogram metric with 12 configured buckets exposes 14 series per label combination: 12 bucket series (`le=...`), one `_sum`, and one `_count`. If that histogram also carries a label like `endpoint` with 50 distinct values, you are already looking at 700 series from one metric family.

## The Two Big Challenges: Size and Cardinality

### Scrape Size

Every scrape is an HTTP response body that Prometheus has to download, parse, and ingest. For small services exposing a few dozen metrics this is trivial, but instrumented runtimes (Go, JVM, .NET) can expose hundreds of metrics by default, and larger applications may expose thousands. Scrape payloads regularly reach several megabytes, and in extreme cases tens of megabytes.

A large scrape causes two concrete problems. First, it consumes bandwidth on every scrape interval (typically 15–60 seconds). Second, and more critically, Prometheus enforces a scrape timeout: if the target takes too long to respond, the scrape is marked as failed and the data is simply not ingested. I encountered scrapes exceeding 64 MB that consistently timed out, meaning that data was *never* in Prometheus, making it impossible to debug the problem from inside Prometheus itself.

One common remedy is enabling gzip compression on the `/metrics` endpoint. Prometheus supports `Accept-Encoding: gzip` out of the box, and the text format compresses well. Compression can reduce transfer size by 80–90%, which helps significantly with bandwidth and timeout margins. However, gzip only addresses the *transport* problem. The data still has to be decompressed and parsed by Prometheus, and all those series still have to be stored and indexed. The real cost of a large scrape is not the bytes on the wire: it is the *cardinality*.

### High Cardinality

Cardinality is the number of distinct time series a metric produces (i.e. the number of unique label value combinations). A metric with no labels has a cardinality of 1. A metric with a `method` label (say, 5 values) and a `status` label (say, 10 values) has a cardinality of up to 50.

Prometheus stores each series independently: it allocates memory for it, writes it to disk, and indexes it. High cardinality therefore translates directly into high memory usage, large on-disk storage, and slower queries. Unlike scrape size, this cost cannot be compressed away. It is structural.

#### Scenario 1: The High-Cardinality Label

The classic example is instrumenting a metric with a label whose value comes from an unbounded domain. Imagine tracking HTTP requests per session:

```
# TYPE http_requests_total counter
http_requests_total{session_id="a1b2c3"} 12
http_requests_total{session_id="x9y8z7"} 4
http_requests_total{session_id="p0q1r2"} 31
...
```

Each new user session creates a new series. With thousands of active sessions, Prometheus is ingesting thousands of new series every minute. Because counters are monotonically increasing, Prometheus keeps these series in memory until they are explicitly garbage-collected, which by default only happens after a series has not been seen for 5 minutes. A busy service with short sessions can create a continuously growing "cardinality debt" that eventually causes Prometheus to run out of memory.

Session IDs are an obvious case, but the same pattern appears with any high-churn identifier: request IDs, trace IDs, user IDs in a large system, or dynamically generated job names.

#### Scenario 2: Label Leaks from Stale Metadata

A similar problem arises when label values can become stale, and the service fails to clean up its own metric registry. A common example in Kubernetes is a service that tracks metrics per pod (perhaps a controller, a proxy, or a sidecar that monitors its neighbors):

```
# TYPE watched_pod_restarts_total counter
watched_pod_restarts_total{pod="web-7d4f9b-xkqzp"} 2
watched_pod_restarts_total{pod="web-7d4f9b-mnprt"} 0
watched_pod_restarts_total{pod="web-7d4f9b-tz9vw"} 5
...
```

When a pod is removed (due to a rolling deployment, a crash, or a scale-down), the correct behavior is to also delete the corresponding metric series from the registry. If the code forgets to do that, the series for the old pod keeps appearing in every scrape indefinitely. The service is essentially accumulating a series for every pod it has ever seen.

This is a pure instrumentation bug, and it can be surprisingly hard to notice. The service appears healthy, the scrape succeeds, and individual series look reasonable in isolation. But over time, especially in clusters with frequent deployments, the cardinality of that metric grows without bound. By the time someone notices the Prometheus memory usage climbing, hundreds or thousands of ghost series may already be present in the scrape.

## Ad-Hoc Analysis: The Shell Toolbox

### Why Not Just Query Prometheus?

Before reaching for shell tools, it is worth asking: why analyse the raw scrape at all, rather than using PromQL inside Prometheus?

There are two situations where Prometheus itself cannot help you. The first is when the scrape never made it into Prometheus in the first place. A scrape that exceeds the configured timeout is simply dropped: no data is ingested, and there is nothing to query. Large scrapes fail exactly like this, which means the tool you would normally use to investigate the problem is blind to it.

The second situation arises when a remote write intermediary sits between the scraper and the TSDB. Tools like [vmagent](https://docs.victoriametrics.com/vmagent/) scrape targets and forward metrics via the remote write protocol, which carries only sample data: it discards `# HELP` and `# TYPE` metadata. Once the data is in the storage backend, you lose the ability to filter or group by metric type, or to see the human-readable descriptions that often give the clearest clue about what a metric is and why its cardinality is high.

In both cases, working directly with the raw scrape text is the only option.

### Counting Series with Shell Tools

Given a saved scrape, a first instinct is to reach for standard Unix tools. Here are the kinds of questions you might try to answer and how you would approach them.

**How many lines does the scrape have?**

```bash
wc -l < prometheus-scrape.txt
```

```
1069
```

**Which metric families are present?**

```bash
grep '^# TYPE' prometheus-scrape.txt
```

```
# TYPE go_gc_cycles_automatic_gc_cycles_total counter
# TYPE go_gc_cycles_forced_gc_cycles_total counter
# TYPE go_gc_cycles_total_gc_cycles_total counter
# TYPE go_gc_duration_seconds summary
# TYPE go_gc_gogc_percent gauge
# TYPE go_gc_gomemlimit_bytes gauge
# TYPE go_gc_heap_allocs_by_size_bytes histogram
...
# TYPE prometheus_http_requests_total counter
# TYPE prometheus_http_response_size_bytes histogram
# TYPE promhttp_metric_handler_requests_in_flight gauge
# TYPE promhttp_metric_handler_requests_total counter
```

**How many series does each metric expose?**

The idea is to count non-comment, non-empty lines per metric family. One approach: strip comment and blank lines, extract the metric name (the part before `{` or the first space), then count occurrences.

```bash
grep -v '^#' prometheus-scrape.txt \
  | grep -v '^$' \
  | sed 's/[{ ].*//' \
  | sort \
  | uniq -c \
  | sort -rn \
  | head -20
```

```
59 prometheus_http_requests_total
20 prometheus_engine_query_duration_histogram_seconds_bucket
18 prometheus_sd_kubernetes_events_total
15 prometheus_tsdb_compaction_duration_seconds_bucket
13 prometheus_tsdb_compaction_chunk_size_bytes_bucket
13 prometheus_tsdb_compaction_chunk_samples_bucket
12 prometheus_engine_query_duration_seconds
12 net_conntrack_dialer_conn_failed_total
12 go_gc_heap_frees_by_size_bytes_bucket
12 go_gc_heap_allocs_by_size_bytes_bucket
11 prometheus_tsdb_compaction_chunk_range_seconds_bucket
10 prometheus_http_request_duration_seconds_bucket
 9 prometheus_http_response_size_bytes_bucket
 8 prometheus_tsdb_sample_ooo_delta_bucket
 8 go_sched_pauses_total_other_seconds_bucket
 8 go_sched_pauses_total_gc_seconds_bucket
 8 go_sched_pauses_stopping_other_seconds_bucket
 8 go_sched_pauses_stopping_gc_seconds_bucket
 8 go_sched_latencies_seconds_bucket
 8 go_gc_pauses_seconds_bucket
```

This pipes the scrape through a sequence of filters: drop comment lines, drop blank lines, strip everything after the metric name, sort, count duplicates, and sort by count descending.

### The Limits of This Approach

The pipeline above works for simple gauges and counters, but it breaks down for histograms and summaries. You can already see this in the output above: `prometheus_engine_query_duration_histogram_seconds_bucket` appears as its own entry with 20 lines, but the corresponding `_sum` and `_count` lines are counted separately and buried lower in the list. The `sed` pattern extracts different "names" for each suffix (`_bucket`, `_sum`, `_count`), so the count is fragmented across multiple rows rather than attributed to a single metric family. The true series count for that histogram is higher than any individual row suggests. Reassembling these correctly requires knowing the metric type, which means parsing the `# TYPE` lines and correlating them with the data lines. That turns a one-liner into a non-trivial script.

There are further edge cases: metrics with no labels, metric names that are prefixes of other metric names, and histograms where the bucket count varies across label dimensions. Shell pipelines are quick to write but fragile to maintain, and getting an accurate cardinality figure for a real-world scrape is harder than it looks.

## Introducing scrapecli

[scrapecli](https://github.com/FRosner/scrapecli) is a small command-line tool that reads a Prometheus scrape from stdin and prints a structured summary. It uses the official Prometheus client libraries to parse the exposition format, so it understands metric types correctly: histograms and summaries are counted as single families, not fragmented by suffix.

### Installation

If you have Go installed:

```bash
go install github.com/FRosner/scrapecli@latest
```

Alternatively, download a pre-built binary for your platform from the [releases page](https://github.com/FRosner/scrapecli/releases).

### Basic Usage

Pipe any Prometheus scrape into scrapecli:

```bash
curl -s localhost:9090/metrics | scrapecli
```

Or analyse a saved scrape file:

```bash
cat prometheus-scrape.txt | scrapecli
```

Running it against the same Prometheus scrape from the previous section produces:

![](https://dev-to-uploads.s3.amazonaws.com/uploads/articles/q7r62zsp4dafhehg9zdv.png)

### What the Output Tells You

**Summary** gives an immediate sense of the scrape's overall footprint: total size on disk and which metrics are consuming the most series. Contrast the Top Metrics list with the awk output from the previous section: `prometheus_engine_query_duration_histogram_seconds` is correctly listed as a single family with 20 series, rather than appearing as a fragmented `_bucket` entry. Each entry also shows its byte contribution, making it easy to see which metrics dominate the scrape size.

**Types** breaks down the metric count by type. Seeing 20 histograms in a scrape is a prompt to check their bucket counts and label cardinality, since histograms multiply series quickly.

**Labels** shows every label name that appears across the scrape, how many distinct values it takes globally, and how many metric families use it. The `handler` label having 59 distinct values across 3 metrics immediately explains why `prometheus_http_requests_total` leads the cardinality ranking. The `<none>` entry counts metrics that carry no labels at all, which is useful context for understanding what fraction of the scrape is label-free.

**Metrics** lists every metric family with its type, series count, labels, and description. This is where you can quickly scan for unfamiliar metrics, check whether a metric's description matches your expectation, or spot a metric with an unexpectedly high cardinality.

### JSON Output for Scripting

Passing `-o json` emits the same information as structured JSON, which is useful for feeding into other tools or automating cardinality checks in CI:

```bash
cat prometheus-scrape.txt | scrapecli -o json
```

```json
{
  "summary": {
    "bytes": 79033,
    "top_cardinalities": [
      { "name": "prometheus_http_requests_total", "cardinality": 59 },
      { "name": "prometheus_engine_query_duration_histogram_seconds", "cardinality": 20 },
      ...
    ],
    "type_counts": {
      "counter": 109,
      "gauge": 90,
      "histogram": 20,
      "summary": 11
    },
    ...
  },
  "metrics": [...]
}
```

You could, for example, use `jq` to fail a CI step if any single metric exceeds a cardinality threshold:

```bash
cat prometheus-scrape.txt | scrapecli -o json \
  | jq '[.summary.top_cardinalities[] | select(.cardinality > 100)] | length > 0'
```

## Conclusion

High cardinality is one of the most common and costly problems in Prometheus deployments, and it tends to be discovered late, usually when memory usage is already climbing or scrapes are already failing. The root cause is usually straightforward once you can see it: an unbounded label, a metric registry that was never cleaned up, a histogram with too many dimensions. The difficulty is getting a clear view of what is actually in a scrape before things go wrong.

Shell tools can get you part of the way there, but they require careful construction and give inaccurate results for histograms and summaries. Querying Prometheus directly is not always an option, especially when the scrape is too large to ingest or when metadata has been stripped by a remote write pipeline.

scrapecli is a small focused tool for exactly this gap: give it a scrape, and it tells you the size, the cardinality leaders, the type breakdown, and the label landscape, and it gets the numbers right. If you maintain a service that exposes Prometheus metrics, it is worth keeping in your toolbox for those moments when you need to understand what your `/metrics` endpoint is actually producing.

The project is open source and available at [github.com/FRosner/scrapecli](https://github.com/FRosner/scrapecli). Have you run into oversized scrapes or runaway cardinality in your own setup? I'd love to hear about it in the comments: what caused it, how you found it, and how you fixed it.
