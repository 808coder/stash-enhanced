package scene

import (
	"testing"
	"time"

	"github.com/stashapp/stash/pkg/file"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type testDestroyConfig struct {
	recordTombstones bool
}

func (c testDestroyConfig) GetVideoFileNamingAlgorithm() models.HashAlgorithm {
	return models.HashAlgorithmOshash
}

func (c testDestroyConfig) GetTombstonesRecord() bool {
	return c.recordTombstones
}

const (
	destroySceneID     = 10
	destroyFileID      = models.FileID(20)
	destroyStudioID    = 30
	destroyPerformerID = 40
	destroyTagID       = 50
)

var (
	destroySceneCreatedAt = time.Date(2024, 3, 4, 5, 6, 7, 0, time.UTC)
	destroyPhash          = int64(-1234567890123)
)

func newDestroyService(db *mocks.Database, record bool) *Service {
	return &Service{
		File:             db.File,
		Repository:       db.Scene,
		MarkerRepository: db.SceneMarker,
		Tombstone:        db.Tombstone,
		Performer:        db.Performer,
		Studio:           db.Studio,
		Tag:              db.Tag,
		Config:           testDestroyConfig{recordTombstones: record},
	}
}

func newDestroyScene() *models.Scene {
	rating := 80
	date := models.DateFromYear(2023)
	studioID := destroyStudioID
	zipID := models.FileID(99) // inside a zip: never touches the filesystem

	f := &models.VideoFile{
		BaseFile: &models.BaseFile{
			ID:       destroyFileID,
			Basename: "clip.mp4",
			Path:     "/videos/clip.mp4",
			Size:     1000,
			DirEntry: models.DirEntry{ZipFileID: &zipID},
			Fingerprints: models.Fingerprints{
				{Type: models.FingerprintTypeOshash, Fingerprint: "oshash1"},
				{Type: models.FingerprintTypeMD5, Fingerprint: "md5one"},
				{Type: models.FingerprintTypePhash, Fingerprint: destroyPhash},
			},
		},
		Duration:   12.5,
		Width:      1920,
		Height:     1080,
		VideoCodec: "h264",
		BitRate:    5000,
	}

	return &models.Scene{
		ID:           destroySceneID,
		Title:        "Translated title",
		Code:         "ABC-123",
		Details:      "Translated details",
		Director:     "not stored",
		Date:         &date,
		Rating:       &rating,
		StudioID:     &studioID,
		CreatedAt:    destroySceneCreatedAt,
		URLs:         models.NewRelatedStrings([]string{"https://example.com/scene"}),
		StashIDs:     models.NewRelatedStashIDs([]models.StashID{{Endpoint: "https://stashdb.org", StashID: "scene-stash-id"}}),
		PerformerIDs: models.NewRelatedIDs([]int{destroyPerformerID}),
		TagIDs:       models.NewRelatedIDs([]int{destroyTagID}),
		Files:        models.NewRelatedVideoFiles([]*models.VideoFile{f}),
	}
}

// expectRowRemoval sets up the calls that remove the scene and its file rows.
// order records the sequence of Tombstone creation and scene row removal.
func expectRowRemoval(db *mocks.Database, deleteFile bool, order *[]string) {
	db.SceneMarker.On("FindBySceneID", testCtx, destroySceneID).Return(nil, nil).Once()
	if deleteFile {
		db.Scene.On("FindByFileID", testCtx, destroyFileID).Return([]*models.Scene{{ID: destroySceneID}}, nil).Once()
		db.File.On("Destroy", testCtx, destroyFileID).Return(nil).Once()
	}
	db.Scene.On("Destroy", testCtx, destroySceneID).Run(func(mock.Arguments) {
		*order = append(*order, "scene destroyed")
	}).Return(nil).Once()
}

// expectSnapshotReads sets up the reads needed to build the snapshot.
func expectSnapshotReads(db *mocks.Database) {
	db.Scene.On("GetCustomFields", testCtx, destroySceneID).Return(map[string]interface{}{
		"original_title":   "Original title",
		"original_details": "Original details",
		"unrelated":        "ignored",
	}, nil).Once()
	db.Scene.On("GetOCount", testCtx, destroySceneID).Return(3, nil).Once()
	db.Scene.On("CountViews", testCtx, destroySceneID).Return(7, nil).Once()

	db.Studio.On("Find", testCtx, destroyStudioID).Return(&models.Studio{ID: destroyStudioID, Name: "Studio A"}, nil).Once()
	db.Studio.On("GetStashIDs", testCtx, destroyStudioID).Return([]models.StashID{
		{Endpoint: "https://stashdb.org", StashID: "studio-stash-id", UpdatedAt: time.Now()},
	}, nil).Once()

	db.Performer.On("FindMany", testCtx, []int{destroyPerformerID}).Return([]*models.Performer{{ID: destroyPerformerID, Name: "Performer A"}}, nil).Once()
	db.Performer.On("GetStashIDs", testCtx, destroyPerformerID).Return([]models.StashID{
		{Endpoint: "https://stashdb.org", StashID: "performer-stash-id"},
	}, nil).Once()

	db.Tag.On("FindMany", testCtx, []int{destroyTagID}).Return([]*models.Tag{{ID: destroyTagID, Name: "Tag A"}}, nil).Once()
}

func expectedTombstone(origin models.TombstoneOrigin, reason string) *models.Tombstone {
	rating := 80
	date := models.DateFromYear(2023)
	phash := destroyPhash
	stashdb := "https://stashdb.org"

	return &models.Tombstone{
		Origin:          origin,
		Reason:          reason,
		SceneID:         destroySceneID,
		Title:           "Translated title",
		Code:            "ABC-123",
		Date:            &date,
		URLs:            []string{"https://example.com/scene"},
		StashIDs:        []models.TombstoneStashID{{Endpoint: stashdb, StashID: "scene-stash-id"}},
		OriginalTitle:   "Original title",
		OriginalDetails: "Original details",
		Details:         "Translated details",
		Rating:          &rating,
		OCounter:        3,
		PlayCount:       7,
		SceneCreatedAt:  destroySceneCreatedAt,
		Studio: &models.TombstoneEntity{
			Name:     "Studio A",
			StashIDs: []models.TombstoneStashID{{Endpoint: stashdb, StashID: "studio-stash-id"}},
		},
		Performers: []models.TombstoneEntity{{
			Name:     "Performer A",
			StashIDs: []models.TombstoneStashID{{Endpoint: stashdb, StashID: "performer-stash-id"}},
		}},
		Tags: []string{"Tag A"},
		Files: []*models.TombstoneFile{{
			Basename:   "clip.mp4",
			Path:       "/videos/clip.mp4",
			Size:       1000,
			Duration:   12.5,
			Width:      1920,
			Height:     1080,
			VideoCodec: "h264",
			BitRate:    5000,
			Oshash:     "oshash1",
			MD5:        "md5one",
			Phash:      &phash,
		}},
	}
}

func newTestFileDeleter() *FileDeleter {
	return &FileDeleter{Deleter: file.NewDeleter()}
}

func TestService_Destroy_writesTombstone(t *testing.T) {
	tests := []struct {
		name   string
		opts   DestroyOptions
		origin models.TombstoneOrigin
		reason string
	}{
		{
			"manual destroy with file deletion",
			DestroyOptions{DeleteFile: true, Tombstone: &TombstoneOptions{Origin: models.TombstoneOriginManual, Reason: "bad quality"}},
			models.TombstoneOriginManual,
			"bad quality",
		},
		{
			"clean task",
			DestroyOptions{Tombstone: &TombstoneOptions{Origin: models.TombstoneOriginClean}},
			models.TombstoneOriginClean,
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := mocks.NewDatabase()
			var order []string
			expectSnapshotReads(db)
			expectRowRemoval(db, tt.opts.DeleteFile, &order)

			var created *models.Tombstone
			db.Tombstone.On("Create", testCtx, mock.AnythingOfType("*models.Tombstone")).Run(func(args mock.Arguments) {
				created = args.Get(1).(*models.Tombstone)
				order = append(order, "tombstone created")
			}).Return(nil).Once()

			s := newDestroyService(db, true)
			err := s.Destroy(testCtx, newDestroyScene(), newTestFileDeleter(), tt.opts)

			assert.NoError(t, err)
			assert.Equal(t, expectedTombstone(tt.origin, tt.reason), created)
			assert.Equal(t, []string{"tombstone created", "scene destroyed"}, order)
			db.AssertExpectations(t)
		})
	}
}

func TestService_Destroy_noTombstone(t *testing.T) {
	tests := []struct {
		name   string
		opts   DestroyOptions
		record bool
	}{
		{
			"database-only destroy",
			DestroyOptions{DestroyFileEntry: false, Tombstone: &TombstoneOptions{Origin: models.TombstoneOriginManual, Reason: "x"}},
			true,
		},
		{
			"opted out, as merge does",
			DestroyOptions{DeleteFile: true},
			true,
		},
		{
			"setting off, manual",
			DestroyOptions{DeleteFile: true, Tombstone: &TombstoneOptions{Origin: models.TombstoneOriginManual}},
			false,
		},
		{
			"setting off, clean",
			DestroyOptions{Tombstone: &TombstoneOptions{Origin: models.TombstoneOriginClean}},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := mocks.NewDatabase()
			var order []string
			expectRowRemoval(db, tt.opts.DeleteFile, &order)

			s := newDestroyService(db, tt.record)
			err := s.Destroy(testCtx, newDestroyScene(), newTestFileDeleter(), tt.opts)

			assert.NoError(t, err)
			assert.Equal(t, []string{"scene destroyed"}, order)
			db.Tombstone.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
			db.AssertExpectations(t)
		})
	}
}

func TestService_Destroy_cleanTombstoneListsFilesRemovedEarlier(t *testing.T) {
	db := mocks.NewDatabase()
	var order []string
	expectSnapshotReads(db)
	expectRowRemoval(db, false, &order)

	var created *models.Tombstone
	db.Tombstone.On("Create", testCtx, mock.AnythingOfType("*models.Tombstone")).Run(func(args mock.Arguments) {
		created = args.Get(1).(*models.Tombstone)
	}).Return(nil).Once()

	earlier := &models.VideoFile{
		BaseFile: &models.BaseFile{
			ID:       destroyFileID + 1,
			Basename: "clip-part1.mp4",
			Path:     "/videos/clip-part1.mp4",
			Size:     500,
			Fingerprints: models.Fingerprints{
				{Type: models.FingerprintTypeOshash, Fingerprint: "oshash0"},
			},
		},
		Duration: 6,
	}

	s := newDestroyService(db, true)
	err := s.Destroy(testCtx, newDestroyScene(), newTestFileDeleter(), DestroyOptions{
		Tombstone: &TombstoneOptions{Origin: models.TombstoneOriginClean, RemovedFiles: []*models.VideoFile{earlier}},
	})

	assert.NoError(t, err)
	want := expectedTombstone(models.TombstoneOriginClean, "")
	want.Files = append([]*models.TombstoneFile{{
		Basename: "clip-part1.mp4",
		Path:     "/videos/clip-part1.mp4",
		Size:     500,
		Duration: 6,
		Oshash:   "oshash0",
	}}, want.Files...)
	assert.Equal(t, want, created)
	db.AssertExpectations(t)
}
