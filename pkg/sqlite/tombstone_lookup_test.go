//go:build integration
// +build integration

package sqlite_test

// These tests stay in pkg/sqlite although they exercise pkg/scene: they need
// real performers, stash IDs and names from the database, and the
// pkg/scene tests have no database setup.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/scene"
)

const lookupEndpoint = "https://stashdb.org/graphql"

func createLookupPerformer(t *testing.T, ctx context.Context, name string, stashIDs ...models.StashID) *models.Performer {
	t.Helper()
	p := &models.Performer{Name: name, StashIDs: models.NewRelatedStashIDs(stashIDs)}
	require.NoError(t, db.Performer.Create(ctx, &models.CreatePerformerInput{Performer: p}))
	return p
}

func lookupPerformerID(t *testing.T, ctx context.Context, frozen models.TombstonePerformer) int {
	t.Helper()
	got, err := scene.FindTombstonePerformer(ctx, db.Performer, frozen)
	require.NoError(t, err)
	if got == nil {
		return 0
	}
	return got.ID
}

func TestFindTombstonePerformer_ByStashID(t *testing.T) {
	runWithRollbackTxn(t, "stash id", func(t *testing.T, ctx context.Context) {
		p := createLookupPerformer(t, ctx, "Renamed performer", models.StashID{Endpoint: lookupEndpoint, StashID: "lookup-uuid"})

		got := lookupPerformerID(t, ctx, models.TombstonePerformer{
			Name: "Old name",
			StashIDs: []models.TombstoneStashID{
				{Endpoint: lookupEndpoint, StashID: "unknown-uuid"},
				{Endpoint: lookupEndpoint, StashID: "lookup-uuid"},
			},
		})

		assert.Equal(t, p.ID, got)
	})
}

func TestFindTombstonePerformer_ByFormerIDWithSameName(t *testing.T) {
	runWithRollbackTxn(t, "former id", func(t *testing.T, ctx context.Context) {
		p := createLookupPerformer(t, ctx, "Former id performer")
		// a second performer with the same name rules out the name step
		twin := &models.Performer{Name: "Former id performer", Disambiguation: "twin"}
		require.NoError(t, db.Performer.Create(ctx, &models.CreatePerformerInput{Performer: twin}))

		got := lookupPerformerID(t, ctx, models.TombstonePerformer{ID: p.ID, Name: "Former id performer"})

		assert.Equal(t, p.ID, got)
	})
}

func TestFindTombstonePerformer_ByUniqueNameIgnoringCase(t *testing.T) {
	runWithRollbackTxn(t, "unique name", func(t *testing.T, ctx context.Context) {
		p := createLookupPerformer(t, ctx, "Recreated Performer")

		got := lookupPerformerID(t, ctx, models.TombstonePerformer{ID: -1, Name: "recreated performer"})

		assert.Equal(t, p.ID, got)
	})
}

func TestFindTombstonePerformer_AmbiguousNameDoesNotMatch(t *testing.T) {
	runWithRollbackTxn(t, "ambiguous name", func(t *testing.T, ctx context.Context) {
		createLookupPerformer(t, ctx, "Common name")
		twin := &models.Performer{Name: "Common name", Disambiguation: "twin"}
		require.NoError(t, db.Performer.Create(ctx, &models.CreatePerformerInput{Performer: twin}))

		got := lookupPerformerID(t, ctx, models.TombstonePerformer{Name: "Common name"})

		assert.Zero(t, got)
	})
}

// A former id now used by a different performer does not match.
func TestFindTombstonePerformer_ReusedIDDoesNotMatch(t *testing.T) {
	runWithRollbackTxn(t, "reused id", func(t *testing.T, ctx context.Context) {
		other := createLookupPerformer(t, ctx, "Someone else")

		got := lookupPerformerID(t, ctx, models.TombstonePerformer{ID: other.ID, Name: "Deleted performer"})

		assert.Zero(t, got)
	})
}
