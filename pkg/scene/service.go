// Package scene provides the application logic for scene functionality.
// Most functionality is provided by [Service].
package scene

import (
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/paths"
	"github.com/stashapp/stash/pkg/plugin"
)

type Config interface {
	GetVideoFileNamingAlgorithm() models.HashAlgorithm
	// GetTombstonesRecord reports whether Tombstones are recorded when
	// scenes are destroyed.
	GetTombstonesRecord() bool
}

type Service struct {
	File             models.FileReaderWriter
	Repository       models.SceneReaderWriter
	MarkerRepository models.SceneMarkerReaderWriter
	PluginCache      *plugin.Cache

	// Tombstone and the readers below are used to snapshot a scene into a
	// Tombstone when it is destroyed.
	Tombstone models.TombstoneWriter
	Performer models.PerformerReader
	Studio    models.StudioReader
	Tag       models.TagReader

	Paths  *paths.Paths
	Config Config
}
