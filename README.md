# scrapecli

## Description

Small command line interface to analyze and interact with Prometheus scrapes

## Usage

```bash
go build
curl -s localhost:9090/metrics | ./scrapecli
```

## Test Resources

- [`prometheus-scrape.txt`](test-resources/prometheus-scrape.txt): `docker run -p 9090:9090 prom/prometheus` and `curl localhost:9090/metrics > prometheus-scrape.txt`
