# Git conventions

Commits, PR titles, branch names and issue titles follow [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/). PR titles are checked by [`.github/workflows/pr-title.yml`](../../.github/workflows/pr-title.yml), which also labels the PR for the release notes. Decision record: issue #27.

All of it is in English.

## Format

```
<type>[(<scope>)][!]: <subject>

[body]

[Refs: #<issue>]
[BREAKING CHANGE: <what breaks>]
```

- **Subject**: imperative, starts lowercase, no trailing period. Keep the header at 72 characters or less.
- **`!`** / **`BREAKING CHANGE:`** mark a breaking change. They only affect the release notes. Versions stay `vX.Y.Z-enhanced.N` (see [`docs/CI.md`](../CI.md)).
- **`Refs: #N`** links a commit to its issue. Don't put issue numbers in the header.

## Types

| Type | Use for | PR label | Release notes |
|---|---|---|---|
| `feat` | new user-facing functionality | `enhancement` | 🚀 Features |
| `fix` | bug fix | `bug` | 🐛 Bug Fixes |
| `perf` | performance improvement | `perf` | ⚡ Performance |
| `refactor` | code change without behaviour change | `refactor` | 🔧 Refactoring |
| `docs` | documentation only | `documentation` | 📝 Documentation |
| `build` | build system, Makefile, Dockerfile | `build` | 📦 Other Changes |
| `revert` | reverts an earlier change | `revert` | 📦 Other Changes |
| `chore` | maintenance, upstream merges | `chore` | 📦 Other Changes |
| `ci` | `.github/workflows/` | `ci` | hidden |
| `test` | tests only | `test` | hidden |
| `style` | formatting, no code change | `style` | hidden |

A `!` adds the `breaking` label, which lists the PR under ⚠️ Breaking Changes.

## Scopes

Optional. Only these are allowed; the PR title check rejects any other:

| Scope | Area |
|---|---|
| `ui` | frontend (`ui/v2.5`) |
| `api` | GraphQL schema and resolvers |
| `db` | SQLite, repositories, migrations |
| `scan` | scanning, identify, tasks |
| `autotag` | auto-tagging |
| `video` | ffmpeg: transcoding, streaming, previews, thumbnails |
| `scraper` | scrapers and stash-box |
| `plugin` | plugin system |
| `config` | settings and configuration (backend and settings UI) |
| `agent` | stash-agent (#9) |
| `ci` | workflows |
| `docker` | Docker images |
| `deps` | dependency updates |
| `upstream` | upstream merges |

To add a scope, add it here **and** to `scopes` in `pr-title.yml` in the same PR.

## Branches

`{type}/{issue-id}_{name}` off `develop`, for example `feat/26_file-info-filename`. Create the issue first to get the id.

Upstream merges: `chore/upstream-vX.Y.Z`, no issue id.

## Pull requests

- Target `develop`.
- Title: the Conventional Commits header, for example `feat(ui): show filename below path in file info tabs`. No issue number; GitHub appends the PR number on merge.
- Body: `Closes #N` for the issue it resolves.
- Merge:
  - feature/fix PRs: **Squash and merge**. The squash commit is the PR title plus the PR description, so the commits inside the PR don't need to be clean.
  - upstream merges: **Create a merge commit**. Squashing would drop the upstream ancestry and make the next upstream merge conflict everywhere.
- Rebase merging is disabled. Merge commits are titled with the PR title too.
- The `PR Title` check is required on `develop`.

## Issues

Issue titles use the same header format, so the title can become the PR title: `feat(agent): scan remote library paths`.
