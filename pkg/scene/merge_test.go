package scene

import (
	"testing"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mergeTestScene returns a scene with all relationships loaded, so Merge
// reads nothing but the calls set up in the test.
func mergeTestScene(id int) *models.Scene {
	return &models.Scene{
		ID:           id,
		Title:        "scene",
		URLs:         models.NewRelatedStrings([]string{}),
		GalleryIDs:   models.NewRelatedIDs([]int{}),
		PerformerIDs: models.NewRelatedIDs([]int{}),
		TagIDs:       models.NewRelatedIDs([]int{}),
		Groups:       models.NewRelatedGroups([]models.GroupsScenes{}),
		StashIDs:     models.NewRelatedStashIDs([]models.StashID{}),
		Files:        models.NewRelatedVideoFiles([]*models.VideoFile{}),
	}
}

func TestService_Merge_writesNoTombstone(t *testing.T) {
	const (
		destID = 1
		srcID  = 2
	)

	db := mocks.NewDatabase()
	db.Scene.On("Find", testCtx, destID).Return(mergeTestScene(destID), nil).Once()
	db.Scene.On("FindMany", testCtx, []int{srcID}).Return([]*models.Scene{mergeTestScene(srcID)}, nil).Once()
	db.SceneMarker.On("FindBySceneID", testCtx, srcID).Return(nil, nil)
	db.Scene.On("UpdatePartial", testCtx, destID, mock.Anything).Return(mergeTestScene(destID), nil).Once()
	db.Scene.On("Destroy", testCtx, srcID).Return(nil).Once()

	// recording is on: only Merge's opt-out keeps the Tombstone away
	s := newDestroyService(db, true)
	err := s.Merge(testCtx, []int{srcID}, destID, newTestFileDeleter(), MergeOptions{ScenePartial: models.NewScenePartial()})

	assert.NoError(t, err)
	db.Tombstone.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	db.AssertExpectations(t)
}
