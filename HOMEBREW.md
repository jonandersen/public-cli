# Homebrew Distribution

This document explains how to set up Homebrew tap distribution for `pub`.

## For Users

### Installation

Once a release is published, users can install `pub` via Homebrew:

```bash
# Install directly (auto-taps the repository)
brew install jonandersen/tap/pub

# Or explicitly tap first
brew tap jonandersen/tap
brew install pub
```

### Upgrading

```bash
brew upgrade pub
```

### Uninstalling

```bash
brew uninstall pub
brew untap jonandersen/tap  # optional
```

## For Maintainers

### Prerequisites

1. **Create the Homebrew tap repository**

   Create a new GitHub repository named `homebrew-tap` under the `jonandersen` account.
   The `homebrew-` prefix is required by Homebrew conventions.

   ```bash
   # On GitHub, create: github.com/jonandersen/homebrew-tap
   # Then initialize it:
   git clone https://github.com/jonandersen/homebrew-tap.git
   cd homebrew-tap
   mkdir Formula
   touch Formula/.gitkeep
   git add .
   git commit -m "Initialize homebrew tap"
   git push
   ```

2. **Create a Personal Access Token (PAT)**

   GoReleaser needs a token to push formula updates to the tap repository.

   - Go to GitHub → Settings → Developer settings → Personal access tokens → Tokens (classic)
   - Click "Generate new token (classic)"
   - Name: `HOMEBREW_TAP_GITHUB_TOKEN`
   - Scopes: `repo` (full control of private repositories)
   - Generate and copy the token

3. **Add the token as a repository secret**

   In the `pub` repository:
   - Go to Settings → Secrets and variables → Actions
   - Click "New repository secret"
   - Name: `HOMEBREW_TAP_GITHUB_TOKEN`
   - Value: paste the PAT from step 2

### Creating a Release

1. **Tag a new version**

   ```bash
   git tag v0.1.0
   git push origin v0.1.0
   ```

2. **GitHub Actions handles the rest**

   The release workflow (`.github/workflows/release.yml`) will:
   - Build binaries for all platforms (linux/darwin, amd64/arm64)
   - Create a GitHub release with the binaries
   - Generate and push the Homebrew formula to `homebrew-tap`

### Testing Locally

Before pushing a release, test GoReleaser locally:

```bash
# Install goreleaser
brew install goreleaser

# Test the configuration (no publish)
goreleaser check
goreleaser release --snapshot --clean

# Check the generated formula
cat dist/homebrew/Formula/pub.rb
```

### Manual Formula Updates

If you need to manually update the formula (not recommended):

1. Build a release tarball
2. Calculate SHA256: `shasum -a 256 pub_*.tar.gz`
3. Update `Formula/pub.rb` in the tap repository

## Configuration Files

| File | Purpose |
|------|---------|
| `.goreleaser.yaml` | GoReleaser configuration for builds and formula generation |
| `.github/workflows/release.yml` | GitHub Actions workflow triggered on version tags |
| `.github/workflows/ci.yml` | CI workflow for PRs (test, lint, build) |
| `Formula/pub.rb` | Template formula (actual formula lives in tap repo) |

## Troubleshooting

### "tap not found" error

Ensure the `homebrew-tap` repository exists and is public.

### Formula not updating

Check that:
- `HOMEBREW_TAP_GITHUB_TOKEN` secret is set
- The token has `repo` scope
- The tap repository allows pushes from the token owner

### Build failures

Run `goreleaser check` to validate the configuration.
Check `.goreleaser.yaml` for syntax errors.

## References

- [GoReleaser Homebrew Documentation](https://goreleaser.com/customization/homebrew/)
- [Homebrew Tap Documentation](https://docs.brew.sh/How-to-Create-and-Maintain-a-Tap)
- [Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
