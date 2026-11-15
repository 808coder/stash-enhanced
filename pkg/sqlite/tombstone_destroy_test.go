//go:build integration
// +build integration

package sqlite_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/scene"
)

type tombstoneDestroyConfig struct{}

func (tombstoneDestroyConfig) GetVideoFileNamingAlgorithm() models.HashAlgorithm {
	return models.HashAlgorithmOshash
}

func (tombstoneDestroyConfig) GetTombstonesRecord() bool { return true }

// The Tombstone of a scene destroyed through the scene service keeps its
// studio, performers and tags after those are deleted.
func TestTombstone_SceneDestroySnapshotIsFrozen(t *testing.T) {
	runWithRollbackTxn(t, "destroy then delete related", func(t *testing.T, ctx context.Context) {
		const stashdb = "https://stashdb.org/graphql"

		studio := &models.CreateStudioInput{Studio: &models.Studio{
			Name:     "Frozen destroy studio",
			StashIDs: models.NewRelatedStashIDs([]models.StashID{{Endpoint: stashdb, StashID: "studio-uuid"}}),
		}}
		require.NoError(t, db.Studio.Create(ctx, studio))
		performer := &models.CreatePerformerInput{Performer: &models.Performer{
			Name:     "Frozen destroy performer",
			StashIDs: models.NewRelatedStashIDs([]models.StashID{{Endpoint: stashdb, StashID: "performer-uuid"}}),
		}}
		require.NoError(t, db.Performer.Create(ctx, performer))
		tag := &models.CreateTagInput{Tag: &models.Tag{Name: "Frozen destroy tag"}}
		require.NoError(t, db.Tag.Create(ctx, tag))

		s := &models.Scene{
			Title:        "Frozen destroy scene",
			StudioID:     &studio.ID,
			PerformerIDs: models.NewRelatedIDs([]int{performer.ID}),
			TagIDs:       models.NewRelatedIDs([]int{tag.ID}),
			URLs:         models.NewRelatedStrings([]string{"https://example.com/frozen"}),
		}
		require.NoError(t, db.Scene.Create(ctx, s, nil))
		require.NoError(t, db.Scene.SetCustomFields(ctx, s.ID, models.CustomFieldsInput{
			Full: map[string]interface{}{models.CustomFieldOriginalTitle: "Original frozen title"},
		}))

		svc := &scene.Service{
			File:             db.File,
			Repository:       db.Scene,
			MarkerRepository: db.SceneMarker,
			Tombstone:        db.Tombstone,
			Performer:        db.Performer,
			Studio:           db.Studio,
			Tag:              db.Tag,
			Config:           tombstoneDestroyConfig{},
		}
		require.NoError(t, svc.Destroy(ctx, s, &scene.FileDeleter{}, scene.DestroyOptions{
			Tombstone: &scene.TombstoneOptions{Origin: models.TombstoneOriginClean},
		}))

		found, err := db.Tombstone.Query(ctx, &models.TombstoneFilter{Title: "Frozen destroy scene"}, nil)
		require.NoError(t, err)
		require.Len(t, found.Matches, 1)
		before := found.Matches[0].Tombstone

		assert.Equal(t, s.ID, before.SceneID)
		assert.Equal(t, models.TombstoneOriginClean, before.Origin)
		assert.Equal(t, "Original frozen title", before.OriginalTitle)
		assert.Equal(t, []string{"https://example.com/frozen"}, before.URLs)
		assert.Equal(t, &models.TombstoneEntity{
			Name:     "Frozen destroy studio",
			StashIDs: []models.TombstoneStashID{{Endpoint: stashdb, StashID: "studio-uuid"}},
		}, before.Studio)
		assert.Equal(t, []models.TombstoneEntity{{
			Name:     "Frozen destroy performer",
			StashIDs: []models.TombstoneStashID{{Endpoint: stashdb, StashID: "performer-uuid"}},
		}}, before.Performers)
		assert.Equal(t, []string{"Frozen destroy tag"}, before.Tags)

		require.NoError(t, db.Performer.Destroy(ctx, performer.ID))
		require.NoError(t, db.Studio.Destroy(ctx, studio.ID))
		require.NoError(t, db.Tag.Destroy(ctx, tag.ID))

		after, err := db.Tombstone.Find(ctx, before.ID)
		require.NoError(t, err)
		assert.Equal(t, before, after)
	})
}
