package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/mocks"
)

func TestTombstoneBasenameTerm(t *testing.T) {
	videoExts := []string{"mp4", "mkv", "avi"}

	tests := []struct {
		name string
		term string
		want string
	}{
		{"video extension is stripped", "clip.mp4", "clip"},
		{"extension case is ignored", "Clip.MKV", "Clip"},
		{"resolution token is stripped", "BangVan.26.10.20.Kathy.Deep.XXX.1080p", "BangVan.26.10.20.Kathy.Deep.XXX"},
		{"extension then resolution", "BangVan.26.10.20.Kathy.Deep.XXX.720P.mkv", "BangVan.26.10.20.Kathy.Deep.XXX"},
		{"extension only is searched as given", ".mkv", ".mkv"},
		{"other extensions are kept", "clip.txt", "clip.txt"},
		{"name with a dot is kept", "Mr. Smith", "Mr. Smith"},
		{"dated release name is kept", "Studio.24.01.01", "Studio.24.01.01"},
		{"last part not an extension", "Kathy.Deep", "Kathy.Deep"},
		{"no dot", "clip", "clip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tombstoneBasenameTerm(tt.term, videoExts))
		})
	}

	t.Run("configured extensions with a dot", func(t *testing.T) {
		assert.Equal(t, "clip", tombstoneBasenameTerm("clip.mp4", []string{".MP4"}))
	})
}

func TestTombstoneUpdate_TrimsReason(t *testing.T) {
	db := mocks.NewDatabase()
	r := newResolver(db)

	const id = 7
	want := &models.Tombstone{ID: id, Reason: "Duplicate"}
	db.Tombstone.On("UpdateReason", mock.Anything, id, "Duplicate").Return(nil).Once()
	db.Tombstone.On("Find", mock.Anything, id).Return(want, nil).Once()

	reason := "  Duplicate \n"
	got, err := r.Mutation().TombstoneUpdate(testCtx, TombstoneUpdateInput{ID: "7", Reason: &reason})
	require.NoError(t, err)
	assert.Equal(t, want, got)

	db.AssertExpectations(t)
}
