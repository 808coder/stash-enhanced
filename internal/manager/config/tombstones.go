package config

import (
	"errors"
	"strings"
)

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
	// TombstonesReasons is the list of selectable deletion reasons.
	TombstonesReasons = "tombstones.reasons"
)

// defaultTombstonesReasons are used when no reasons are configured.
var defaultTombstonesReasons = []string{
	"Not my taste",
	"Low quality",
	"Replaced by a better version",
	"Duplicate",
	"Broken file",
}

// GetTombstonesReasons returns the selectable deletion reasons, in order.
// Falls back to a default list when unset.
func (i *Config) GetTombstonesReasons() []string {
	i.RLock()
	defer i.RUnlock()

	v := i.forKey(TombstonesReasons)
	if !v.Exists(TombstonesReasons) {
		return append([]string(nil), defaultTombstonesReasons...)
	}
	return v.Strings(TombstonesReasons)
}

// ErrNoTombstoneReasons is returned when a reasons list is empty after
// normalising: at least one deletion reason is required.
var ErrNoTombstoneReasons = errors.New("at least one deletion reason is required")

// NormaliseTombstoneReasons trims each entry, drops empty entries and drops
// case-insensitive duplicates, keeping the first. It returns
// ErrNoTombstoneReasons if nothing is left.
func NormaliseTombstoneReasons(in []string) ([]string, error) {
	seen := make(map[string]struct{}, len(in))
	ret := make([]string, 0, len(in))
	for _, r := range in {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		k := strings.ToLower(r)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		ret = append(ret, r)
	}
	if len(ret) == 0 {
		return nil, ErrNoTombstoneReasons
	}
	return ret, nil
}

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
