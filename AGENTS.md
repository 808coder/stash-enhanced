## Agent skills

### Issue tracker

GitHub Issues in `808coder/stash-enhanced`, via `gh`. See `docs/agents/issue-tracker.md`.

### Triage labels

Default five triage labels (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`), plus the GitHub default labels. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: `GLOSSARY.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.

## Architecture

See `docs/ARCHITECTURE.md` for the backend layering, GraphQL request flow, and frontend structure. Read it before changing resolvers, repositories, or migrations.

## CI/CD

See `docs/CI.md` for the fork's pipeline (`fork-build.yml`), release/versioning scheme, and which upstream workflows are disabled. Read it before touching `.github/workflows/` or cutting a release. `gh` defaults to upstream here: always pass `-R 808coder/stash-enhanced`.
