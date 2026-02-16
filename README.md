# scrapecli

## Description

Small command line interface to analyze and interact with Prometheus scrapes

## Installation

### Using Homebrew

If you have Homebrew installed on macOS or Linux, you can install scrapecli via our tap:

```bash
brew tap FRosner/tap
brew install scrapecli
```

### Using go install

If you have Go installed, you can install the latest version directly:

```bash
go install github.com/FRosner/scrapecli@latest
```

### From Release

Download the latest release for your platform from the [releases page](https://github.com/FRosner/scrapecli/releases).

### From Source

```bash
go build
```

## Usage

```bash
curl -s localhost:9090/metrics | scrapecli
```
<img width="709" height="1217" alt="image" src="https://github.com/user-attachments/assets/4f605c22-0b1b-436b-a9a9-2577aaa9c7dc" />

If you built from source, use `./scrapecli` instead.

## Releasing

To create a new release:

### Option 1: Create and push a tag (Automatic)

1. Create and push a new tag with a `v` prefix:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

### Option 2: Manually trigger the workflow (Testing)

1. Go to the [Actions tab](https://github.com/FRosner/scrapecli/actions/workflows/release.yml)
2. Click "Run workflow"
3. Enter the tag name (e.g., `v1.0.0`)
4. Click "Run workflow" button

The workflow will automatically create the tag and trigger the release process.

### What happens during a release:

The GitHub Actions workflow will automatically:
   - Build binaries for multiple platforms (Linux, macOS, Windows)
   - Create archives (tar.gz and zip)
   - Generate checksums
   - Create a GitHub release with all artifacts
   - Update the Homebrew tap at [FRosner/homebrew-tap](https://github.com/FRosner/homebrew-tap) with the new formula

### Homebrew Tap

The Homebrew tap is automatically maintained by GoReleaser. When a new release is created, GoReleaser will:
- Generate a Homebrew formula based on the release artifacts
- Push the formula to the [FRosner/homebrew-tap](https://github.com/FRosner/homebrew-tap) repository
- Users can then install scrapecli via `brew tap FRosner/tap && brew install scrapecli`

Note: The tap repository will be created automatically on the first release.

### Testing the Homebrew Installation

After creating a release, you can test the Homebrew installation:

1. **Add the tap:**
   ```bash
   brew tap FRosner/tap
   ```

2. **Install scrapecli:**
   ```bash
   brew install scrapecli
   ```

3. **Verify the installation:**
   ```bash
   scrapecli --version
   # or test with actual metrics
   curl -s localhost:9090/metrics | scrapecli
   ```

4. **Uninstall (if needed):**
   ```bash
   brew uninstall scrapecli
   brew untap FRosner/tap
   ```

## Test Resources

- [`prometheus-scrape.txt`](test-resources/prometheus-scrape.txt): `docker run -p 9090:9090 prom/prometheus` and `curl localhost:9090/metrics > prometheus-scrape.txt`
