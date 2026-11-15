package scene

import (
	"context"
	"testing"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func tombstoneTestFile() *models.VideoFile {
	return &models.VideoFile{
		BaseFile: &models.BaseFile{
			ID:   1,
			Path: "test.mp4",
			Fingerprints: models.Fingerprints{
				{Type: models.FingerprintTypeOshash, Fingerprint: "abc"},
				{Type: models.FingerprintTypeMD5, Fingerprint: "def"},
			},
		},
	}
}

func newTombstoneHandler(db *mocks.Database, record bool, tagID string) *ScanHandler {
	return &ScanHandler{
		TombstoneStore:         db.Tombstone,
		TombstoneTags:          db.Tag,
		RecordTombstones:       record,
		PreviouslyDeletedTagID: tagID,
	}
}

func TestApplyTombstones(t *testing.T) {
	manual := &models.Tombstone{ID: 10, Origin: models.TombstoneOriginManual}
	clean := &models.Tombstone{ID: 11, Origin: models.TombstoneOriginClean}

	t.Run("manual tombstone adds configured tag and keeps tombstone", func(t *testing.T) {
		db := mocks.NewDatabase()
		h := newTombstoneHandler(db, true, "5")
		db.Tombstone.On("FindByFileFingerprints", mock.Anything, "abc", "def").Return([]*models.Tombstone{manual}, nil)
		db.Tag.On("Find", mock.Anything, 5).Return(&models.Tag{ID: 5}, nil)

		s := models.NewScene()
		db.WithTxnCtx(func(ctx context.Context) {
			assert.NoError(t, h.applyTombstones(ctx, &s, tombstoneTestFile()))
		})
		assert.Equal(t, []int{5}, s.TagIDs.List())
		db.Tombstone.AssertNotCalled(t, "Destroy", mock.Anything, mock.Anything)
	})

	t.Run("manual tombstone without configured tag finds Previously deleted", func(t *testing.T) {
		db := mocks.NewDatabase()
		h := newTombstoneHandler(db, true, "")
		db.Tombstone.On("FindByFileFingerprints", mock.Anything, "abc", "def").Return([]*models.Tombstone{manual}, nil)
		db.Tag.On("FindByName", mock.Anything, "Previously deleted", true).Return(&models.Tag{ID: 7}, nil)

		s := models.NewScene()
		db.WithTxnCtx(func(ctx context.Context) {
			assert.NoError(t, h.applyTombstones(ctx, &s, tombstoneTestFile()))
		})
		assert.Equal(t, []int{7}, s.TagIDs.List())
		db.Tag.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("manual tombstone creates missing fallback tag", func(t *testing.T) {
		db := mocks.NewDatabase()
		h := newTombstoneHandler(db, true, "")
		db.Tombstone.On("FindByFileFingerprints", mock.Anything, "abc", "def").Return([]*models.Tombstone{manual}, nil)
		db.Tag.On("FindByName", mock.Anything, "Previously deleted", true).Return(nil, nil)
		db.Tag.On("Create", mock.Anything, mock.AnythingOfType("*models.CreateTagInput")).
			Run(func(args mock.Arguments) {
				args.Get(1).(*models.CreateTagInput).Tag.ID = 9
			}).Return(nil)

		s := models.NewScene()
		db.WithTxnCtx(func(ctx context.Context) {
			assert.NoError(t, h.applyTombstones(ctx, &s, tombstoneTestFile()))
		})
		assert.Equal(t, []int{9}, s.TagIDs.List())
	})

	t.Run("clean tombstone is removed without tag", func(t *testing.T) {
		db := mocks.NewDatabase()
		h := newTombstoneHandler(db, true, "5")
		db.Tombstone.On("FindByFileFingerprints", mock.Anything, "abc", "def").Return([]*models.Tombstone{clean}, nil)
		db.Tombstone.On("Destroy", mock.Anything, 11).Return(nil)

		s := models.NewScene()
		db.WithTxnCtx(func(ctx context.Context) {
			assert.NoError(t, h.applyTombstones(ctx, &s, tombstoneTestFile()))
		})
		assert.False(t, s.TagIDs.Loaded())
		db.Tombstone.AssertCalled(t, "Destroy", mock.Anything, 11)
	})

	t.Run("manual and clean: tag added, only clean removed", func(t *testing.T) {
		db := mocks.NewDatabase()
		h := newTombstoneHandler(db, true, "5")
		db.Tombstone.On("FindByFileFingerprints", mock.Anything, "abc", "def").Return([]*models.Tombstone{manual, clean}, nil)
		db.Tombstone.On("Destroy", mock.Anything, 11).Return(nil)
		db.Tag.On("Find", mock.Anything, 5).Return(&models.Tag{ID: 5}, nil)

		s := models.NewScene()
		db.WithTxnCtx(func(ctx context.Context) {
			assert.NoError(t, h.applyTombstones(ctx, &s, tombstoneTestFile()))
		})
		assert.Equal(t, []int{5}, s.TagIDs.List())
		db.Tombstone.AssertNotCalled(t, "Destroy", mock.Anything, 10)
	})

	t.Run("no tombstone does nothing", func(t *testing.T) {
		db := mocks.NewDatabase()
		h := newTombstoneHandler(db, true, "5")
		db.Tombstone.On("FindByFileFingerprints", mock.Anything, "abc", "def").Return(nil, nil)

		s := models.NewScene()
		db.WithTxnCtx(func(ctx context.Context) {
			assert.NoError(t, h.applyTombstones(ctx, &s, tombstoneTestFile()))
		})
		assert.False(t, s.TagIDs.Loaded())
	})

	t.Run("setting off performs no lookup", func(t *testing.T) {
		db := mocks.NewDatabase()
		h := newTombstoneHandler(db, false, "5")

		s := models.NewScene()
		db.WithTxnCtx(func(ctx context.Context) {
			assert.NoError(t, h.applyTombstones(ctx, &s, tombstoneTestFile()))
		})
		db.Tombstone.AssertNotCalled(t, "FindByFileFingerprints", mock.Anything, mock.Anything, mock.Anything)
		assert.False(t, s.TagIDs.Loaded())
	})
}
