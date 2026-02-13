# scrapecli

## Description

Small command line interface to analyze and interact with Prometheus scrapes

## Installation

### From Release

Download the latest release for your platform from the [releases page](https://github.com/FRosner/scrapecli/releases).

### From Source

```bash
go build
```

## Usage

```bash
curl -s localhost:9090/metrics | ./scrapecli
```

## Releasing

To create a new release:

1. Create and push a new tag with a `v` prefix:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. The GitHub Actions workflow will automatically:
   - Build binaries for multiple platforms (Linux, macOS, Windows)
   - Create archives (tar.gz and zip)
   - Generate checksums
   - Create a GitHub release with all artifacts

## Test Resources

- [`prometheus-scrape.txt`](test-resources/prometheus-scrape.txt): `docker run -p 9090:9090 prom/prometheus` and `curl localhost:9090/metrics > prometheus-scrape.txt`
