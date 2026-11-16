package scene

import "github.com/stashapp/stash/pkg/models"

// CleanTombstones decides, over one run of the Clean task, which clean
// Tombstone to request when a scene is destroyed.
//
// Clean removes a scene's files one at a time and destroys the scene with
// its last file, so the files removed earlier are remembered here to be
// listed in the Tombstone. A Tombstone is only requested when all of the
// scene's files are gone from disk: files that still exist but are excluded
// (.stashignore, library settings, extensions) do not produce one.
//
// The zero value is ready to use. It is not safe for concurrent use.
type CleanTombstones struct {
	removed map[int][]*models.VideoFile
	present map[int]bool
}

// FileRemoved records that Clean removed f from the scene sceneID, which
// keeps other files. missing reports whether f is gone from disk.
func (c *CleanTombstones) FileRemoved(sceneID int, f *models.VideoFile, missing bool) {
	if c.removed == nil {
		c.removed = make(map[int][]*models.VideoFile)
		c.present = make(map[int]bool)
	}

	c.removed[sceneID] = append(c.removed[sceneID], f)
	if !missing {
		c.present[sceneID] = true
	}
}

// ForDestroy returns the Tombstone request for destroying the scene sceneID
// together with its last file. lastMissing reports whether that file is gone
// from disk. It returns nil when any of the scene's files is still on disk.
func (c *CleanTombstones) ForDestroy(sceneID int, lastMissing bool) *TombstoneOptions {
	if !lastMissing || c.present[sceneID] {
		return nil
	}

	return &TombstoneOptions{
		Origin:       models.TombstoneOriginClean,
		RemovedFiles: c.removed[sceneID],
	}
}
