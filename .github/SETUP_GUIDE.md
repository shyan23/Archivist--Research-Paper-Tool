# GitHub CI/CD Setup Guide

This guide explains how to set up the CI/CD pipeline for Archivist on GitHub.

## 🚀 Quick Start

**Good News:** Most of the CI/CD pipeline works **automatically** once you push to GitHub!
**No manual commands or GUI toggles required** for basic functionality.

## What Happens Automatically

Once you push this repository to GitHub:

✅ **CI Pipeline** runs automatically on:
- Every push to `master`, `main`, or `develop` branches
- Every pull request

✅ **Release Pipeline** runs automatically on:
- Every tag push (e.g., `v1.0.0`)

✅ **Dependabot** runs automatically:
- Weekly dependency updates

## Required GitHub Secrets

For full functionality, you need to configure these secrets in your GitHub repository:

### Navigate to: `Settings` → `Secrets and variables` → `Actions` → `New repository secret`

| Secret Name | Required For | How to Get |
|------------|--------------|------------|
| `GEMINI_API_KEY` | Integration tests | Get from [Google AI Studio](https://makersuite.google.com/app/apikey) |
| `CODECOV_TOKEN` | Coverage reporting | Get from [Codecov.io](https://codecov.io) (free for public repos) |
| `DOCKER_USERNAME` | Docker Hub releases | Your Docker Hub username |
| `DOCKER_PASSWORD` | Docker Hub releases | Docker Hub access token |

**Note:** The pipeline will work without these secrets, but some features will be skipped:
- Tests requiring API will be marked as "skipped"
- Coverage reports won't be uploaded
- Docker images won't be pushed to Docker Hub (GitHub Container Registry still works)

## Optional: Enable GitHub Container Registry

To publish Docker images to GitHub Container Registry (ghcr.io):

1. Go to `Settings` → `Actions` → `General`
2. Scroll to "Workflow permissions"
3. Select "Read and write permissions"
4. Check "Allow GitHub Actions to create and approve pull requests"
5. Click "Save"

This is **already enabled by default** in most repositories!

## Optional: Enable Security Scanning

To see security scan results:

1. Go to `Settings` → `Code security and analysis`
2. Enable "Dependency graph" (usually enabled by default)
3. Enable "Dependabot alerts"
4. Enable "Dependabot security updates"
5. Enable "Code scanning" → "Set up" → Use the default CodeQL workflow

## Optional: Enable GitHub Pages (for Documentation)

If you want to publish documentation:

1. Go to `Settings` → `Pages`
2. Source: Select "Deploy from a branch"
3. Branch: Select `gh-pages` and `/root`
4. Click "Save"

## Creating Your First Release

To create a release and trigger the CD pipeline:

```bash
# Tag your commit
git tag -a v1.0.0 -m "Release version 1.0.0"

# Push the tag
git push origin v1.0.0
```

This will automatically:
- Run all tests
- Build binaries for Linux, macOS, Windows (amd64, arm64)
- Create Docker images
- Create a GitHub Release with all artifacts
- Generate changelog

## Viewing CI/CD Status

### In Your Repository:
1. Click the "Actions" tab
2. See all workflow runs
3. Click any run to see detailed logs

### Badge for README:
Add to your README.md:
```markdown
![CI](https://github.com/YOUR_USERNAME/archivist/workflows/CI/badge.svg)
![Release](https://github.com/YOUR_USERNAME/archivist/workflows/Release/badge.svg)
```

## Workflow Files Explained

### `.github/workflows/ci.yml`
**Runs on:** Every push and PR
**Does:**
- Linting (code quality checks)
- Testing (unit + integration tests)
- Building (compiles the binary)
- Docker build
- Security scanning

**Duration:** ~3-5 minutes

### `.github/workflows/release.yml`
**Runs on:** Tag pushes (v*.*.*)
**Does:**
- Build multi-platform binaries
- Create Docker images
- Publish to Docker Hub and GitHub Container Registry
- Create GitHub Release with artifacts

**Duration:** ~5-10 minutes

## Troubleshooting

### "Tests are failing in CI but pass locally"
- Check if GEMINI_API_KEY is set in secrets
- Check if Go version matches (we test on 1.20, 1.21, 1.22)
- Check for race conditions (CI runs with `-race` flag)

### "Docker build fails"
- Ensure Dockerfile is in repository root
- Check Docker build logs in Actions tab
- Verify all required files are included

### "Release workflow doesn't trigger"
- Ensure tag follows `v*.*.*` format (e.g., `v1.0.0`)
- Check if tag was pushed: `git push origin --tags`
- Verify workflow permissions are correct

### "Dependabot PRs are not created"
- Check `.github/dependabot.yml` is present
- Ensure Dependabot is enabled in repository settings
- May take up to 24 hours for first run

## Advanced: Customizing Workflows

### Changing Test Timeout
Edit `.github/workflows/ci.yml`:
```yaml
- name: Run unit tests
  run: go test -v -timeout=10m ./tests/unit/...
```

### Adding More Go Versions
Edit `.github/workflows/ci.yml`:
```yaml
strategy:
  matrix:
    go-version: ['1.20', '1.21', '1.22', '1.23']
```

### Changing Release Platforms
Edit `.goreleaser.yml`:
```yaml
builds:
  - goos:
      - linux
      - windows
      - darwin
      - freebsd  # Add more platforms
```

## Security Best Practices

1. **Never commit secrets** - Always use GitHub Secrets
2. **Use Dependabot** - Automatically updates dependencies
3. **Review security alerts** - Check the Security tab regularly
4. **Use signed commits** - Enable GPG signing for releases
5. **Restrict workflow permissions** - Use least privilege principle

## Monitoring and Metrics

### CodeCov Dashboard
- Visit https://codecov.io/gh/YOUR_USERNAME/archivist
- View coverage trends over time
- See which files need more tests

### GitHub Insights
- Go to `Insights` → `Community`
- Check repository health score
- View contributor statistics

## Support

If you encounter issues:
1. Check the Actions tab for detailed error logs
2. Review this guide
3. Check [GitHub Actions documentation](https://docs.github.com/en/actions)
4. Open an issue in the repository

## Summary Checklist

- [ ] Repository pushed to GitHub
- [ ] Secrets configured (at minimum: GEMINI_API_KEY)
- [ ] First commit pushed to see CI in action
- [ ] Created first tag to test release pipeline
- [ ] Added status badges to README
- [ ] Enabled security features
- [ ] Star the repository! ⭐

---

**Congratulations!** Your CI/CD pipeline is now set up and will run automatically on every commit and release.
