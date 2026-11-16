package models

import "context"

type TombstoneReader interface {
	// Find returns the Tombstone with its files, or nil if not found.
	Find(ctx context.Context, id int) (*Tombstone, error)
	// Query returns the Tombstones matching filter (OR over its keys, AND its
	// Origin), paginated and sorted by findFilter (Page, PerPage, Sort and
	// Direction are used; nil returns the first page of 25, newest first).
	// Sort is one of the TombstoneSort constants; an unknown sort is an
	// error. Each match lists the keys it matched on.
	Query(ctx context.Context, filter *TombstoneFilter, findFilter *FindFilterType) (*TombstoneQueryResult, error)
	// FindByFileFingerprints returns the Tombstones, with their files, that
	// have a file whose oshash or md5 equals the given value. Empty values
	// are ignored.
	FindByFileFingerprints(ctx context.Context, oshash string, md5 string) ([]*Tombstone, error)
}

type TombstoneWriter interface {
	// Create stores newTombstone and its files and sets their IDs. A zero
	// DeletedAt is set to the current time.
	Create(ctx context.Context, newTombstone *Tombstone) error
	// UpdateReason replaces the reason of a Tombstone; empty clears it.
	UpdateReason(ctx context.Context, id int, reason string) error
	// Destroy removes a Tombstone and its files.
	Destroy(ctx context.Context, id int) error
}

type TombstoneReaderWriter interface {
	TombstoneReader
	TombstoneWriter
}
