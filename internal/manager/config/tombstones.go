package config

// Scene tombstone options. Kept in their own file to limit merge conflicts
// with upstream config.go.
const (
	// TombstonesRecord enables recording tombstones for deleted scenes.
	// Defaults to true.
	TombstonesRecord = "tombstones.record"
	// TombstonesPreviouslyDeletedTagID is the id of the tag added to a
	// rescanned scene that was manually deleted before. Empty means a tag
	// named "Previously deleted" is found or created on demand.
	TombstonesPreviouslyDeletedTagID = "tombstones.previously_deleted_tag_id"
)

// GetTombstonesRecord returns true if tombstones are recorded when scenes are
// deleted. Defaults to true.
func (i *Config) GetTombstonesRecord() bool {
	return i.getBoolDefault(TombstonesRecord, true)
}

// GetTombstonesPreviouslyDeletedTagID returns the configured id of the
// "previously deleted" tag, or an empty string if none is configured.
func (i *Config) GetTombstonesPreviouslyDeletedTagID() string {
	return i.getString(TombstonesPreviouslyDeletedTagID)
}
