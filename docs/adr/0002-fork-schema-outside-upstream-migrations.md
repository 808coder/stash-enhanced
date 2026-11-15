# Fork-owned tables outside upstream's migration chain

- Status: accepted
- Date: 2026-11-16
- Issues: #48

## Context

Tombstones (#48) need two new tables in the main SQLite database. Upstream manages the schema with numbered migrations (`pkg/sqlite/migrations/<n>_*.up.sql`) and a single `appSchemaVersion` in `pkg/sqlite/database.go`. The migrator (golang-migrate) stores one version number in `schema_migrations` and only compares numbers: every new migration must be exactly current + 1.

The fork merges upstream regularly. Upstream keeps adding migrations.

## Decision

Fork-owned tables are prefixed `fork_` and created by a **fork schema step** (`pkg/sqlite/fork_schema.go`) that runs on every database open, after upstream migrations:

- It runs from `Database.initialise()`, which is reached from `Open` (existing or new database) and from `ReInitialise` (after the UI migration job). It never runs while an upstream migration is pending, because `Open` returns `MigrationNeededError` before `initialise`.
- It keeps its own version in the table `fork_schema_version` (one row). `forkSchemaSteps` is an append-only list of SQL scripts; step n brings the fork schema to version n. Each step runs in its own transaction together with the version update.
- Every step is idempotent (`CREATE TABLE IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`), so a database that already has the tables is unaffected.
- `appSchemaVersion` and upstream's `schema_migrations` are never touched. Upstream's runner ignores the extra tables.
- Fork tables do not reference upstream tables with foreign keys, so upstream migrations that rebuild a table (create new, copy, drop old) cannot break or cascade into them.

The first step creates `fork_tombstones` and `fork_tombstone_files`.

## Rejected options

- **A numbered upstream-style migration.** It takes the next number in upstream's chain. The next upstream migration with the same number collides on every merge, and renumbering breaks databases that already ran the fork's migration.
- **A separate SQLite file.** Writing the Tombstone would not be atomic with the scene deletion (no shared transaction), and the file would not be part of the normal database backup and restore.

## Consequences

- Upstream merges never conflict on migration numbers or `appSchemaVersion`.
- Backup (`VACUUM INTO` / file copy) and restore include the fork tables automatically.
- The anonymiser has to clear fork tables explicitly; it opens the copy with `Open`, so the schema step runs there as well.
- Later fork schema changes append a step to `forkSchemaSteps`. Steps that alter existing fork tables cannot rely on `IF NOT EXISTS` and must check the current state themselves.
- If upstream ever adds a table with a `fork_` prefix, the names collide. This is unlikely and would show up at merge time.
