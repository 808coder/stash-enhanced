package sqlite

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

// The fork keeps its own tables outside upstream's numbered migration chain
// so that upstream merges never collide on migration numbers. See
// docs/adr/0002-fork-schema-outside-upstream-migrations.md.
//
// forkSchemaSteps are applied in order, once each, on every database open
// after upstream migrations. Step n (1-based) brings the fork schema to
// version n. Every step must be idempotent ("IF NOT EXISTS"), so a step that
// was applied but not recorded can safely run again. Never edit or reorder an
// existing step; append a new one.
var forkSchemaSteps = []string{
	// 1: tombstones (#48)
	`
CREATE TABLE IF NOT EXISTS fork_tombstones (
  id integer not null primary key autoincrement,
  deleted_at datetime not null,
  origin varchar(255) not null,
  reason text,
  scene_id integer,
  title text,
  code text,
  date date,
  date_precision tinyint,
  urls text not null default '[]',
  stash_ids text not null default '[]',
  original_title text,
  original_details text,
  details text,
  rating tinyint,
  o_counter integer not null default 0,
  play_count integer not null default 0,
  scene_created_at datetime,
  studio text,
  performers text not null default '[]',
  tags text not null default '[]'
);
CREATE INDEX IF NOT EXISTS index_fork_tombstones_on_deleted_at ON fork_tombstones (deleted_at);
CREATE INDEX IF NOT EXISTS index_fork_tombstones_on_code ON fork_tombstones (code);
CREATE TABLE IF NOT EXISTS fork_tombstone_files (
  id integer not null primary key autoincrement,
  tombstone_id integer not null,
  basename text not null default '',
  basename_stem text not null default '',
  path text not null default '',
  size integer not null default 0,
  duration float not null default 0,
  width integer not null default 0,
  height integer not null default 0,
  video_codec text not null default '',
  bit_rate integer not null default 0,
  oshash text,
  md5 text,
  phash integer,
  foreign key(tombstone_id) references fork_tombstones(id) on delete cascade
);
CREATE INDEX IF NOT EXISTS index_fork_tombstone_files_on_tombstone_id ON fork_tombstone_files (tombstone_id);
CREATE INDEX IF NOT EXISTS index_fork_tombstone_files_on_oshash ON fork_tombstone_files (oshash);
CREATE INDEX IF NOT EXISTS index_fork_tombstone_files_on_md5 ON fork_tombstone_files (md5);
`,
}

const forkSchemaVersionTable = "fork_schema_version"

// runForkSchema brings the fork-owned tables up to date. It must only run
// after upstream migrations are complete.
func runForkSchema(conn *sqlx.DB) error {
	if _, err := conn.Exec(`CREATE TABLE IF NOT EXISTS ` + forkSchemaVersionTable + ` (version integer not null)`); err != nil {
		return fmt.Errorf("creating fork schema version table: %w", err)
	}

	var version int
	if err := conn.Get(&version, `SELECT COALESCE(MAX(version), 0) FROM `+forkSchemaVersionTable); err != nil {
		return fmt.Errorf("reading fork schema version: %w", err)
	}

	for i := version; i < len(forkSchemaSteps); i++ {
		if err := applyForkSchemaStep(conn, i+1, forkSchemaSteps[i]); err != nil {
			return fmt.Errorf("applying fork schema step %d: %w", i+1, err)
		}
	}

	return nil
}

func applyForkSchemaStep(conn *sqlx.DB, version int, step string) error {
	tx, err := conn.Beginx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(step); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM ` + forkSchemaVersionTable); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO `+forkSchemaVersionTable+` (version) VALUES (?)`, version); err != nil {
		return err
	}

	return tx.Commit()
}
