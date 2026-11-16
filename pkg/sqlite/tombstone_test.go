//go:build integration
// +build integration

package sqlite_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sqlite"
	"github.com/stashapp/stash/pkg/txn"
)

func fullTombstone() *models.Tombstone {
	date, _ := models.ParseDate("2024-03-05")
	phash := int64(-7210000000000000000)
	return &models.Tombstone{
		DeletedAt:       time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		Origin:          models.TombstoneOriginManual,
		Reason:          "bad quality",
		SceneID:         4711,
		Title:           "Translated title",
		Code:            "ABC-123",
		Date:            &date,
		URLs:            []string{"https://example.com/a", "https://example.com/b"},
		StashIDs:        []models.TombstoneStashID{{Endpoint: "https://stashdb.org/graphql", StashID: "scene-uuid"}},
		OriginalTitle:   "Original title",
		OriginalDetails: "Original details",
		Details:         "Translated details",
		Rating:          ptr(80),
		OCounter:        3,
		PlayCount:       7,
		SceneCreatedAt:  time.Date(2020, 5, 6, 7, 8, 9, 0, time.UTC),
		Studio: &models.TombstoneEntity{
			Name:     "Studio name",
			StashIDs: []models.TombstoneStashID{{Endpoint: "https://stashdb.org/graphql", StashID: "studio-uuid"}},
		},
		Performers: []models.TombstonePerformer{
			{ID: 12, Name: "Performer A", Gender: ptr(models.GenderEnumFemale), StashIDs: []models.TombstoneStashID{{Endpoint: "https://stashdb.org/graphql", StashID: "perf-a"}}},
			{Name: "Performer B", StashIDs: []models.TombstoneStashID{}},
		},
		Tags: []string{"Tag A", "Tag B"},
		Files: []*models.TombstoneFile{
			{
				Basename:   "Clip.Name.mp4",
				Path:       "/media/videos/Clip.Name.mp4",
				Size:       123456789,
				Duration:   1234.5,
				Width:      1920,
				Height:     1080,
				VideoCodec: "h264",
				BitRate:    8000000,
				Oshash:     "0123456789abcdef",
				MD5:        "d41d8cd98f00b204e9800998ecf8427e",
				Phash:      &phash,
			},
			{
				Basename: "second.mkv",
				Path:     "/media/videos/second.mkv",
				Oshash:   "fedcba9876543210",
			},
		},
	}
}

func createTombstone(t *testing.T, ctx context.Context, ts *models.Tombstone) *models.Tombstone {
	t.Helper()
	require.NoError(t, db.Tombstone.Create(ctx, ts))
	return ts
}

func TestTombstoneStore_CreateFind(t *testing.T) {
	runWithRollbackTxn(t, "round trip", func(t *testing.T, ctx context.Context) {
		ts := createTombstone(t, ctx, fullTombstone())

		assert.NotZero(t, ts.ID)
		for _, f := range ts.Files {
			assert.NotZero(t, f.ID)
			assert.Equal(t, ts.ID, f.TombstoneID)
		}

		got, err := db.Tombstone.Find(ctx, ts.ID)
		require.NoError(t, err)
		require.NotNil(t, got)

		want := fullTombstone()
		want.ID = ts.ID
		for i := range want.Files {
			want.Files[i].ID = ts.Files[i].ID
			want.Files[i].TombstoneID = ts.ID
		}
		assert.Equal(t, want, got)
	})

	runWithRollbackTxn(t, "minimal tombstone", func(t *testing.T, ctx context.Context) {
		ts := createTombstone(t, ctx, &models.Tombstone{Origin: models.TombstoneOriginClean, SceneID: 1})

		got, err := db.Tombstone.Find(ctx, ts.ID)
		require.NoError(t, err)
		assert.Equal(t, models.TombstoneOriginClean, got.Origin)
		assert.False(t, got.DeletedAt.IsZero(), "deleted at defaults to now")
		assert.Nil(t, got.Date)
		assert.Nil(t, got.Rating)
		assert.Nil(t, got.Studio)
		assert.Empty(t, got.Files)
	})

	runWithRollbackTxn(t, "not found", func(t *testing.T, ctx context.Context) {
		got, err := db.Tombstone.Find(ctx, -1)
		assert.NoError(t, err)
		assert.Nil(t, got)
	})
}

func TestTombstoneStore_FrozenAfterRelatedDeleted(t *testing.T) {
	runWithRollbackTxn(t, "related entities deleted", func(t *testing.T, ctx context.Context) {
		performer := &models.CreatePerformerInput{Performer: &models.Performer{Name: "Frozen performer"}}
		require.NoError(t, db.Performer.Create(ctx, performer))
		studio := &models.CreateStudioInput{Studio: &models.Studio{Name: "Frozen studio"}}
		require.NoError(t, db.Studio.Create(ctx, studio))
		tag := &models.CreateTagInput{Tag: &models.Tag{Name: "Frozen tag"}}
		require.NoError(t, db.Tag.Create(ctx, tag))

		ts := createTombstone(t, ctx, &models.Tombstone{
			Origin:     models.TombstoneOriginManual,
			Studio:     &models.TombstoneEntity{Name: studio.Name},
			Performers: []models.TombstonePerformer{{ID: performer.ID, Name: performer.Name}},
			Tags:       []string{tag.Name},
		})

		require.NoError(t, db.Performer.Destroy(ctx, performer.ID))
		require.NoError(t, db.Studio.Destroy(ctx, studio.ID))
		require.NoError(t, db.Tag.Destroy(ctx, tag.ID))

		got, err := db.Tombstone.Find(ctx, ts.ID)
		require.NoError(t, err)
		assert.Equal(t, "Frozen studio", got.Studio.Name)
		assert.Equal(t, "Frozen performer", got.Performers[0].Name)
		assert.Equal(t, []string{"Frozen tag"}, got.Tags)
	})
}

func TestTombstoneSchema_ReopenIsIdempotent(t *testing.T) {
	f, err := os.CreateTemp("", "*.sqlite")
	require.NoError(t, err)
	f.Close()
	defer os.Remove(f.Name())

	d := sqlite.NewDatabase()
	require.NoError(t, d.Open(f.Name()))
	require.NoError(t, d.Close())

	// second open runs the fork schema step against existing tables
	d = sqlite.NewDatabase()
	require.NoError(t, d.Open(f.Name()))
	defer d.Close()

	// the post-migration path runs the step a third time
	require.NoError(t, d.ReInitialise())

	assert.Equal(t, d.AppSchemaVersion(), d.Version(), "upstream schema version untouched")

	err = withTxnOn(d, func(ctx context.Context) error {
		ts := &models.Tombstone{Origin: models.TombstoneOriginManual}
		if err := d.Tombstone.Create(ctx, ts); err != nil {
			return err
		}
		got, err := d.Tombstone.Find(ctx, ts.ID)
		if err != nil {
			return err
		}
		assert.NotNil(t, got)
		return nil
	})
	assert.NoError(t, err)
}

func withTxnOn(d *sqlite.Database, f func(ctx context.Context) error) error {
	return txn.WithTxn(context.Background(), d, f)
}

// queryFixture stores three Tombstones with distinct keys, oldest first.
type queryFixture struct {
	a, b, c *models.Tombstone
}

const (
	fixturePhashA = int64(0x0f0f0f0f0f0f0f0f)
	// fixturePhashB differs from fixturePhashA in exactly 4 bits
	fixturePhashB = int64(0x0f0f0f0f0f0f0f00)
)

func createQueryFixture(t *testing.T, ctx context.Context) queryFixture {
	t.Helper()
	phashA, phashB := fixturePhashA, fixturePhashB
	return queryFixture{
		a: createTombstone(t, ctx, &models.Tombstone{
			DeletedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			Origin:    models.TombstoneOriginManual,
			Title:     "Summer Holiday",
			Code:      "CODE-A",
			URLs:      []string{"https://example.com/a"},
			StashIDs:  []models.TombstoneStashID{{Endpoint: "https://stashdb.org/graphql", StashID: "stash-a"}},
			Files: []*models.TombstoneFile{
				{Basename: "Holiday.Clip.MP4", Oshash: "oshash-a", MD5: "md5-a", Phash: &phashA},
			},
		}),
		b: createTombstone(t, ctx, &models.Tombstone{
			DeletedAt:     time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			Origin:        models.TombstoneOriginClean,
			Title:         "Translated",
			OriginalTitle: "Winter Original",
			Code:          "CODE-B",
			URLs:          []string{"https://example.com/b"},
			Files: []*models.TombstoneFile{
				{Basename: "first.mkv", Oshash: "oshash-b1"},
				{Basename: "other_name.mkv", Oshash: "oshash-b2", MD5: "md5-b2", Phash: &phashB},
			},
		}),
		c: createTombstone(t, ctx, &models.Tombstone{
			DeletedAt: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			Origin:    models.TombstoneOriginManual,
			Title:     "Unrelated",
			StashIDs:  []models.TombstoneStashID{{Endpoint: "https://other/graphql", StashID: "stash-c"}},
		}),
	}
}

type tombstoneHit struct {
	ID   int
	Keys []models.TombstoneMatchKey
}

func queryHits(t *testing.T, ctx context.Context, filter *models.TombstoneFilter, findFilter *models.FindFilterType) ([]tombstoneHit, int) {
	t.Helper()
	res, err := db.Tombstone.Query(ctx, filter, findFilter)
	require.NoError(t, err)
	hits := []tombstoneHit{}
	for _, m := range res.Matches {
		hits = append(hits, tombstoneHit{ID: m.Tombstone.ID, Keys: m.MatchedKeys})
	}
	return hits, res.Count
}

func keys(k ...models.TombstoneMatchKey) []models.TombstoneMatchKey { return k }

func TestTombstoneStore_QuerySingleKey(t *testing.T) {
	runWithRollbackTxn(t, "single keys", func(t *testing.T, ctx context.Context) {
		fx := createQueryFixture(t, ctx)

		tests := []struct {
			name   string
			filter models.TombstoneFilter
			want   []tombstoneHit
		}{
			{"url", models.TombstoneFilter{URL: "https://example.com/b"},
				[]tombstoneHit{{fx.b.ID, keys(models.TombstoneMatchKeyURL)}}},
			{"url is exact", models.TombstoneFilter{URL: "https://example.com"}, []tombstoneHit{}},
			{"stash id", models.TombstoneFilter{StashID: "stash-c"},
				[]tombstoneHit{{fx.c.ID, keys(models.TombstoneMatchKeyStashID)}}},
			{"oshash of second file", models.TombstoneFilter{Oshash: "oshash-b2"},
				[]tombstoneHit{{fx.b.ID, keys(models.TombstoneMatchKeyOshash)}}},
			{"md5", models.TombstoneFilter{MD5: "md5-a"},
				[]tombstoneHit{{fx.a.ID, keys(models.TombstoneMatchKeyMD5)}}},
			{"code", models.TombstoneFilter{Code: "CODE-A"},
				[]tombstoneHit{{fx.a.ID, keys(models.TombstoneMatchKeyCode)}}},
			{"title substring ignores case", models.TombstoneFilter{Title: "summer hol"},
				[]tombstoneHit{{fx.a.ID, keys(models.TombstoneMatchKeyTitle)}}},
			{"title matches original title", models.TombstoneFilter{Title: "winter"},
				[]tombstoneHit{{fx.b.ID, keys(models.TombstoneMatchKeyTitle)}}},
			{"title wildcards are literal", models.TombstoneFilter{Title: "%"}, []tombstoneHit{}},
			{"filename substring ignores case", models.TombstoneFilter{Basename: "holiday.clip"},
				[]tombstoneHit{{fx.a.ID, keys(models.TombstoneMatchKeyBasename)}}},
			{"filename ignores search extension", models.TombstoneFilter{Basename: "other_name.avi"},
				[]tombstoneHit{{fx.b.ID, keys(models.TombstoneMatchKeyBasename)}}},
			{"filename does not match extension", models.TombstoneFilter{Basename: "mkv"}, []tombstoneHit{}},
			{"phash exact", models.TombstoneFilter{Phash: &models.TombstonePhashInput{Value: fixturePhashA, Distance: 0}},
				[]tombstoneHit{{fx.a.ID, keys(models.TombstoneMatchKeyPhash)}}},
			// like findScenes' phash_distance criterion, the distance is exclusive
			{"phash within distance", models.TombstoneFilter{Phash: &models.TombstonePhashInput{Value: fixturePhashA, Distance: 5}},
				[]tombstoneHit{{fx.b.ID, keys(models.TombstoneMatchKeyPhash)}, {fx.a.ID, keys(models.TombstoneMatchKeyPhash)}}},
			{"phash at distance is excluded", models.TombstoneFilter{Phash: &models.TombstonePhashInput{Value: fixturePhashB, Distance: 4}},
				[]tombstoneHit{{fx.b.ID, keys(models.TombstoneMatchKeyPhash)}}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				filter := tt.filter
				got, count := queryHits(t, ctx, &filter, nil)
				assert.Equal(t, tt.want, got)
				assert.Equal(t, len(tt.want), count)
			})
		}
	})
}

func TestTombstoneStore_QueryCombined(t *testing.T) {
	runWithRollbackTxn(t, "keys combined with or", func(t *testing.T, ctx context.Context) {
		fx := createQueryFixture(t, ctx)

		got, count := queryHits(t, ctx, &models.TombstoneFilter{
			URL:     "https://example.com/a",
			Code:    "CODE-A",
			StashID: "stash-c",
			MD5:     "md5-b2",
			Title:   "nothing matches this",
		}, nil)

		assert.Equal(t, []tombstoneHit{
			{fx.c.ID, keys(models.TombstoneMatchKeyStashID)},
			{fx.b.ID, keys(models.TombstoneMatchKeyMD5)},
			{fx.a.ID, keys(models.TombstoneMatchKeyURL, models.TombstoneMatchKeyCode)},
		}, got)
		assert.Equal(t, 3, count)
	})
}

func TestTombstoneStore_QueryAllPaginated(t *testing.T) {
	runWithRollbackTxn(t, "no filter newest first", func(t *testing.T, ctx context.Context) {
		fx := createQueryFixture(t, ctx)

		got, count := queryHits(t, ctx, nil, nil)
		assert.Equal(t, []tombstoneHit{{fx.c.ID, nil}, {fx.b.ID, nil}, {fx.a.ID, nil}}, got)
		assert.Equal(t, 3, count)

		got, count = queryHits(t, ctx, &models.TombstoneFilter{}, &models.FindFilterType{Page: ptr(2), PerPage: ptr(2)})
		assert.Equal(t, []tombstoneHit{{fx.a.ID, nil}}, got)
		assert.Equal(t, 3, count)

		got, _ = queryHits(t, ctx, nil, &models.FindFilterType{PerPage: ptr(-1)})
		assert.Len(t, got, 3)
	})

	runWithRollbackTxn(t, "results carry files", func(t *testing.T, ctx context.Context) {
		fx := createQueryFixture(t, ctx)

		res, err := db.Tombstone.Query(ctx, &models.TombstoneFilter{Code: "CODE-B"}, nil)
		require.NoError(t, err)
		require.Len(t, res.Matches, 1)
		assert.Equal(t, fx.b, res.Matches[0].Tombstone)
	})
}

func TestTombstoneStore_QueryOrigin(t *testing.T) {
	runWithRollbackTxn(t, "origin restricts the keys", func(t *testing.T, ctx context.Context) {
		fx := createQueryFixture(t, ctx)
		manual, clean := models.TombstoneOriginManual, models.TombstoneOriginClean

		// origin alone is not a key: no matched keys are reported
		res, err := db.Tombstone.Query(ctx, &models.TombstoneFilter{Origin: &clean}, nil)
		require.NoError(t, err)
		require.Len(t, res.Matches, 1)
		assert.Equal(t, fx.b.ID, res.Matches[0].Tombstone.ID)
		assert.Empty(t, res.Matches[0].MatchedKeys)
		assert.Equal(t, 1, res.Count)
		// the counts per origin ignore the origin filter
		assert.Equal(t, 2, res.ManualCount)
		assert.Equal(t, 1, res.CleanCount)

		// keys are ORed, then restricted by origin
		res, err = db.Tombstone.Query(ctx, &models.TombstoneFilter{
			Origin: &manual,
			Code:   "CODE-A",
			MD5:    "md5-b2",
		}, nil)
		require.NoError(t, err)
		require.Len(t, res.Matches, 1)
		assert.Equal(t, fx.a.ID, res.Matches[0].Tombstone.ID)
		assert.Equal(t, keys(models.TombstoneMatchKeyCode), res.Matches[0].MatchedKeys)
		assert.Equal(t, 1, res.Count)
		assert.Equal(t, 1, res.ManualCount)
		assert.Equal(t, 1, res.CleanCount)

		// without an origin the counts add up to the total
		res, err = db.Tombstone.Query(ctx, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 3, res.Count)
		assert.Equal(t, 2, res.ManualCount)
		assert.Equal(t, 1, res.CleanCount)
	})
}

func TestTombstoneStore_QuerySort(t *testing.T) {
	runWithRollbackTxn(t, "sort", func(t *testing.T, ctx context.Context) {
		// phashes sort by their hex string: 0x0f.. before 0xf0..
		phashLow, phashHigh := int64(0x0f00000000000000), int64(-0x1000000000000000)

		// oldest first; x has no title, reason or files
		a := createTombstone(t, ctx, &models.Tombstone{
			DeletedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			Origin:    models.TombstoneOriginManual,
			Title:     "beta",
			Reason:    "Duplicate",
			Files: []*models.TombstoneFile{
				{Basename: "b.mp4", Size: 300, Height: 720, Phash: &phashHigh},
				// only the first file counts
				{Basename: "0.mp4", Size: 1, Height: 1, Phash: &phashLow},
			},
		})
		b := createTombstone(t, ctx, &models.Tombstone{
			DeletedAt:     time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			Origin:        models.TombstoneOriginClean,
			OriginalTitle: "Alpha",
			Reason:        "broken",
			Files:         []*models.TombstoneFile{{Basename: "C.mp4", Size: 100, Height: 2160, Phash: &phashLow}},
		})
		c := createTombstone(t, ctx, &models.Tombstone{
			DeletedAt: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			Origin:    models.TombstoneOriginManual,
			Title:     "Gamma",
			Files:     []*models.TombstoneFile{{Basename: "a.mp4", Size: 200, Height: 1080}},
		})
		x := createTombstone(t, ctx, &models.Tombstone{
			DeletedAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
			Origin:    models.TombstoneOriginManual,
		})

		asc, desc := models.SortDirectionEnumAsc, models.SortDirectionEnumDesc

		tests := []struct {
			sort string
			dir  models.SortDirectionEnum
			want []int
		}{
			{models.TombstoneSortDeletedAt, asc, []int{a.ID, b.ID, c.ID, x.ID}},
			{models.TombstoneSortDeletedAt, desc, []int{x.ID, c.ID, b.ID, a.ID}},
			// case-insensitive, original title as fallback, untitled last
			{models.TombstoneSortTitle, asc, []int{b.ID, a.ID, c.ID, x.ID}},
			{models.TombstoneSortTitle, desc, []int{c.ID, a.ID, b.ID, x.ID}},
			// ties newest first
			{models.TombstoneSortOrigin, asc, []int{b.ID, x.ID, c.ID, a.ID}},
			// no reason last in both directions
			{models.TombstoneSortReason, asc, []int{b.ID, a.ID, x.ID, c.ID}},
			{models.TombstoneSortReason, desc, []int{a.ID, b.ID, x.ID, c.ID}},
			// file sorts use the first file; no file last
			{models.TombstoneSortSize, asc, []int{b.ID, c.ID, a.ID, x.ID}},
			{models.TombstoneSortSize, desc, []int{a.ID, c.ID, b.ID, x.ID}},
			{models.TombstoneSortResolution, desc, []int{b.ID, c.ID, a.ID, x.ID}},
			{models.TombstoneSortBasename, asc, []int{c.ID, a.ID, b.ID, x.ID}},
			{models.TombstoneSortPhash, asc, []int{b.ID, a.ID, x.ID, c.ID}},
			{models.TombstoneSortPhash, desc, []int{a.ID, b.ID, x.ID, c.ID}},
		}

		for _, tt := range tests {
			t.Run(tt.sort+" "+tt.dir.String(), func(t *testing.T) {
				sort, dir := tt.sort, tt.dir
				res, err := db.Tombstone.Query(ctx, nil, &models.FindFilterType{Sort: &sort, Direction: &dir})
				require.NoError(t, err)
				got := []int{}
				for _, m := range res.Matches {
					got = append(got, m.Tombstone.ID)
				}
				assert.Equal(t, tt.want, got)
			})
		}

		t.Run("sorted pages", func(t *testing.T) {
			sort := models.TombstoneSortSize
			res, err := db.Tombstone.Query(ctx, nil, &models.FindFilterType{Sort: &sort, Page: ptr(2), PerPage: ptr(2)})
			require.NoError(t, err)
			require.Len(t, res.Matches, 2)
			assert.Equal(t, a.ID, res.Matches[0].Tombstone.ID)
			assert.Equal(t, x.ID, res.Matches[1].Tombstone.ID)
		})

		t.Run("unknown sort is rejected", func(t *testing.T) {
			sort := "id; DROP TABLE fork_tombstones"
			_, err := db.Tombstone.Query(ctx, nil, &models.FindFilterType{Sort: &sort})
			assert.Error(t, err)
		})
	})
}

func TestTombstoneStore_FindByFileFingerprints(t *testing.T) {
	runWithRollbackTxn(t, "fingerprints", func(t *testing.T, ctx context.Context) {
		fx := createQueryFixture(t, ctx)

		ids := func(ts []*models.Tombstone) []int {
			ret := []int{}
			for _, t := range ts {
				ret = append(ret, t.ID)
			}
			return ret
		}

		got, err := db.Tombstone.FindByFileFingerprints(ctx, "oshash-b1", "")
		require.NoError(t, err)
		assert.Equal(t, []int{fx.b.ID}, ids(got))
		assert.Equal(t, models.TombstoneOriginClean, got[0].Origin)
		assert.Len(t, got[0].Files, 2)

		got, err = db.Tombstone.FindByFileFingerprints(ctx, "", "md5-a")
		require.NoError(t, err)
		assert.Equal(t, []int{fx.a.ID}, ids(got))

		got, err = db.Tombstone.FindByFileFingerprints(ctx, "oshash-a", "md5-b2")
		require.NoError(t, err)
		assert.ElementsMatch(t, []int{fx.a.ID, fx.b.ID}, ids(got))

		got, err = db.Tombstone.FindByFileFingerprints(ctx, "unknown", "")
		require.NoError(t, err)
		assert.Empty(t, got)

		got, err = db.Tombstone.FindByFileFingerprints(ctx, "", "")
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestTombstoneStore_UpdateReasonDestroy(t *testing.T) {
	runWithRollbackTxn(t, "update reason", func(t *testing.T, ctx context.Context) {
		ts := createTombstone(t, ctx, fullTombstone())

		require.NoError(t, db.Tombstone.UpdateReason(ctx, ts.ID, "changed my mind"))
		got, err := db.Tombstone.Find(ctx, ts.ID)
		require.NoError(t, err)
		want := *ts
		want.Reason = "changed my mind"
		assert.Equal(t, &want, got, "only the reason changes")

		require.NoError(t, db.Tombstone.UpdateReason(ctx, ts.ID, ""))
		got, err = db.Tombstone.Find(ctx, ts.ID)
		require.NoError(t, err)
		assert.Equal(t, "", got.Reason)

		assert.Error(t, db.Tombstone.UpdateReason(ctx, -1, "x"))
	})

	runWithRollbackTxn(t, "destroy", func(t *testing.T, ctx context.Context) {
		ts := createTombstone(t, ctx, fullTombstone())
		other := createTombstone(t, ctx, &models.Tombstone{Origin: models.TombstoneOriginManual})

		require.NoError(t, db.Tombstone.Destroy(ctx, ts.ID))

		got, err := db.Tombstone.Find(ctx, ts.ID)
		require.NoError(t, err)
		assert.Nil(t, got)

		found, err := db.Tombstone.FindByFileFingerprints(ctx, ts.Files[0].Oshash, "")
		require.NoError(t, err)
		assert.Empty(t, found, "files are removed with the tombstone")

		got, err = db.Tombstone.Find(ctx, other.ID)
		require.NoError(t, err)
		assert.NotNil(t, got)

		assert.Error(t, db.Tombstone.Destroy(ctx, ts.ID), "destroying a missing tombstone fails")
	})
}

func TestTombstoneAnonymiser_RemovesTombstones(t *testing.T) {
	var ts *models.Tombstone
	require.NoError(t, withTxn(func(ctx context.Context) error {
		ts = fullTombstone()
		return db.Tombstone.Create(ctx, ts)
	}))
	t.Cleanup(func() {
		_ = withTxn(func(ctx context.Context) error { return db.Tombstone.Destroy(ctx, ts.ID) })
	})

	f, err := os.CreateTemp("", "*.sqlite")
	require.NoError(t, err)
	f.Close()
	defer os.Remove(f.Name())

	anonymiser, err := sqlite.NewAnonymiser(db, f.Name())
	require.NoError(t, err)
	require.NoError(t, anonymiser.Anonymise(context.Background()))

	out := sqlite.NewDatabase()
	require.NoError(t, out.Open(f.Name()))
	defer out.Close()

	require.NoError(t, txn.WithReadTxn(context.Background(), out, func(ctx context.Context) error {
		res, err := out.Tombstone.Query(ctx, nil, &models.FindFilterType{PerPage: ptr(-1)})
		if err != nil {
			return err
		}
		assert.Zero(t, res.Count, "no tombstones in anonymised database")

		found, err := out.Tombstone.FindByFileFingerprints(ctx, ts.Files[0].Oshash, ts.Files[0].MD5)
		if err != nil {
			return err
		}
		assert.Empty(t, found, "no tombstone files in anonymised database")
		return nil
	}))
}
