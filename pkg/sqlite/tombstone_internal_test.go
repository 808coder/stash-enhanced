package sqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stashapp/stash/pkg/models"
)

// Performers stored before gender and the former id were recorded still
// decode, with neither set.
func TestTombstoneRow_DecodesPerformersWithoutGenderOrID(t *testing.T) {
	r := tombstoneRow{
		ID:         1,
		Origin:     string(models.TombstoneOriginManual),
		Performers: `[{"name":"Old performer","stash_ids":[{"endpoint":"https://stashdb.org/graphql","stash_id":"old-uuid"}]}]`,
	}

	got, err := r.resolve()
	require.NoError(t, err)
	assert.Equal(t, []models.TombstonePerformer{{
		Name:     "Old performer",
		StashIDs: []models.TombstoneStashID{{Endpoint: "https://stashdb.org/graphql", StashID: "old-uuid"}},
	}}, got.Performers)
}

func TestTombstoneSorts_CoverEveryKey(t *testing.T) {
	for _, key := range models.AllTombstoneSort {
		assert.Contains(t, tombstoneSorts, key)
	}
	assert.Len(t, tombstoneSorts, len(models.AllTombstoneSort))
}
