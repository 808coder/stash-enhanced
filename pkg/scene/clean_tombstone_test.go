package scene

import (
	"testing"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stretchr/testify/assert"
)

func cleanTestFile(id models.FileID) *models.VideoFile {
	return &models.VideoFile{BaseFile: &models.BaseFile{ID: id}}
}

func TestCleanTombstones_ForDestroy(t *testing.T) {
	const sceneID = 1
	clean := models.TombstoneOriginClean

	t.Run("last file gone from disk", func(t *testing.T) {
		var c CleanTombstones
		assert.Equal(t, &TombstoneOptions{Origin: clean}, c.ForDestroy(sceneID, true))
	})

	t.Run("last file excluded but still on disk", func(t *testing.T) {
		var c CleanTombstones
		assert.Nil(t, c.ForDestroy(sceneID, false))
	})

	t.Run("multi-file scene, all files gone", func(t *testing.T) {
		var c CleanTombstones
		f1, f2 := cleanTestFile(1), cleanTestFile(2)
		c.FileRemoved(sceneID, f1, true)
		c.FileRemoved(sceneID, f2, true)
		c.FileRemoved(sceneID+1, cleanTestFile(9), true)

		assert.Equal(t, &TombstoneOptions{Origin: clean, RemovedFiles: []*models.VideoFile{f1, f2}}, c.ForDestroy(sceneID, true))
	})

	t.Run("multi-file scene, an earlier file still on disk", func(t *testing.T) {
		var c CleanTombstones
		c.FileRemoved(sceneID, cleanTestFile(1), false)

		assert.Nil(t, c.ForDestroy(sceneID, true))
	})
}
