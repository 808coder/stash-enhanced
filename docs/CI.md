# CI/CD

This fork builds, tests, releases and publishes Docker images with [`.github/workflows/fork-build.yml`](../.github/workflows/fork-build.yml). Decision record: issue #23.

## Workflows

| Workflow | File | State |
|---|---|---|
| Fork Build | `fork-build.yml` | active, the fork's pipeline |
| PR Title | `pr-title.yml` | active, checks Conventional Commits PR titles and labels PRs by type, see [`agents/git-conventions.md`](agents/git-conventions.md) |
| Lint (golangci-lint) | `golangci-lint.yml` | active, unchanged from upstream |
| Build | `build.yml` | **disabled**, upstream pipeline |
| Compiler Build | `build-compiler.yml` | **disabled**, pushes to `ghcr.io/stashapp/compiler` (no access) |

The upstream files are left unchanged so that upstream merges don't conflict, and are disabled in GitHub instead:

```bash
gh workflow disable "Build" -R 808coder/stash-enhanced
gh workflow disable "Compiler Build" -R 808coder/stash-enhanced
```

After every upstream merge, run `gh workflow list -R 808coder/stash-enhanced --all` and disable any **new** upstream workflow that shouldn't run here.

> `gh` in this clone defaults to upstream `stashapp/stash`. Always pass `-R 808coder/stash-enhanced`.

## What runs when

| Event | Build + test | Docker image | GitHub release |
|---|---|---|---|
| PR into `develop` | yes | built + smoke test, not pushed | none |
| Push to `develop` | yes | built + smoke test, not pushed | none |
| Release `v*` published | yes | `:latest` + `:<tag>` | assets uploaded to that release |

Jobs: `generate` (Go codegen + UI), then `test` and `build` in parallel, then `docker` and `release` (releases only).

Binaries from any run (including `develop` pushes) can be downloaded from the run's artifacts in the Actions tab for 1 day.

`OFFICIAL_BUILD` is `true` for everything except PRs. `UPDATE_REPO` is `808coder/stash-enhanced`, so the built-in update check queries this fork's releases.

## Versioning and tags

- Releases are tagged `vX.Y.Z-enhanced.N` directly on `develop`. `X.Y.Z` is the upstream version this release is based on, and `N` restarts at `1` after each upstream merge.
- Upstream tags are not kept in this fork. The `upstream` remote must not fetch them:
  ```bash
  git config remote.upstream.tagopt --no-tags
  ```
- The version comes from `git describe --tags --exclude latest_develop`. Builds between releases look like `v0.32.0-enhanced.1-5-gabc1234`. The app treats those as develop builds, whose update check looks for a `latest_develop` release. This fork doesn't publish one, so the update check only works for release builds (exact tags, which use `/releases/latest`).
- Only tag `v*` for real releases.

## Cutting a release

```bash
git switch develop && git pull
git tag -a v0.32.0-enhanced.2 -m "v0.32.0-enhanced.2"
git push origin v0.32.0-enhanced.2
gh release create v0.32.0-enhanced.2 --generate-notes --verify-tag -R 808coder/stash-enhanced
```

Release notes are generated from PR labels via [`.github/release.yml`](../.github/release.yml). `pr-title.yml` sets those labels from the PR title's Conventional Commits type, so the notes are grouped into Breaking Changes, Features, Bug Fixes, Performance, Refactoring, Documentation and Other Changes; `ci`, `test` and `style` PRs are left out.

A `release` event runs the workflow **from the tagged commit**, so the tag must point at a commit that already contains `fork-build.yml`.

The workflow marks the release as non-prerelease and latest, because the update check reads `/releases/latest`, which skips prereleases.

## Secrets

| Secret | Value |
|---|---|
| `DOCKERHUB_USERNAME` | `808coder` |
| `DOCKERHUB_TOKEN` | Docker Hub personal access token, Read & Write |

Tags and releases use the built-in `GITHUB_TOKEN`; the `release` job requests `contents: write`.

## Adding a platform

Only `linux/amd64` is built today. It is built natively on `ubuntu-24.04` with CGO (`make build-cc-linux`, static-pie, so the binary has no glibc dependency and runs in the Alpine image).

To add a target, for example Windows:

1. Add a matrix entry to the `build` job:
   ```yaml
   - platform: windows
     make-target: build-cc-windows
     container: true          # cross-compilers come from ghcr.io/stashapp/compiler
     artifact-paths: |
       dist/stash-win.exe
   ```
   `container: false` only works for the runner's own architecture (amd64).
2. Append the binary to `RELEASE_FILES` in the `release` job.
3. For a new Linux architecture in the Docker image, also add it to `platforms` in the `docker` job's push step. `docker/ci/x86_64/Dockerfile` already maps `linux/arm64`, `linux/arm/v7` and `linux/arm/v6` to their binaries. The `docker` job then needs those binaries in `dist/`, and the smoke-test build keeps `linux/amd64` only.

The available `build-cc-*` targets are listed in the [`Makefile`](../Makefile).

## Not covered yet

- `stash-agent` Docker image (comes with a slice of #9)
- Automatic deployment to the LXC/NAS
