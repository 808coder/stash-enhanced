//go:build integration
// +build integration

package translation

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/plugin/hook"
	"github.com/stashapp/stash/pkg/sqlite"
	"github.com/stashapp/stash/pkg/translate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	// necessary to register custom migrations
	_ "github.com/stashapp/stash/pkg/sqlite/migrations"
)

const (
	endpointA = "https://stashdb.org/graphql"
	endpointB = "https://fansdb.cc/graphql"
)

var tagTestOptions = Options{
	LanguageMappings: map[int]string{1000: "ja", 1001: "ru"},
	TargetLanguage:   "en",
	MergeOnCollision: true,
}

func newTestRepository(t *testing.T) models.Repository {
	t.Helper()
	db := sqlite.NewDatabase()
	require.NoError(t, db.Open(filepath.Join(t.TempDir(), "stash.sqlite")))
	t.Cleanup(func() { _ = db.Close() })
	return db.Repository()
}

// dictTranslator translates from a fixed dictionary.
type dictTranslator struct {
	dict  map[string]string
	texts []translate.Text
}

func (d *dictTranslator) Name() string { return "fake:dict" }

func (d *dictTranslator) Translate(_ context.Context, texts []translate.Text, _, _ string) ([]string, error) {
	ret := make([]string, len(texts))
	for i, t := range texts {
		d.texts = append(d.texts, t)
		ret[i] = d.dict[t.Value]
	}
	return ret, nil
}

type hookCall struct {
	id       int
	hookType hook.TriggerEnum
	input    interface{}
}

type recordingTagHooks struct{ calls []hookCall }

func (h *recordingTagHooks) ExecutePostHooks(_ context.Context, id int, hookType hook.TriggerEnum, input interface{}, _ []string) {
	h.calls = append(h.calls, hookCall{id, hookType, input})
}

type tagSpec struct {
	name        string
	description string
	aliases     []string
	parents     []int
	children    []int
	stashIDs    []models.StashID
}

func createTag(t *testing.T, r models.Repository, s tagSpec) int {
	t.Helper()
	tag := &models.Tag{
		Name:        s.name,
		Description: s.description,
		Aliases:     models.NewRelatedStrings(s.aliases),
		ParentIDs:   models.NewRelatedIDs(s.parents),
		ChildIDs:    models.NewRelatedIDs(s.children),
		StashIDs:    models.NewRelatedStashIDs(s.stashIDs),
	}
	require.NoError(t, r.WithTxn(context.Background(), func(ctx context.Context) error {
		return r.Tag.Create(ctx, &models.CreateTagInput{Tag: tag})
	}))
	return tag.ID
}

func createScene(t *testing.T, r models.Repository, tagIDs ...int) int {
	t.Helper()
	s := &models.Scene{TagIDs: models.NewRelatedIDs(tagIDs)}
	require.NoError(t, r.WithTxn(context.Background(), func(ctx context.Context) error {
		return r.Scene.Create(ctx, s, nil)
	}))
	return s.ID
}

type tagState struct {
	tag      *models.Tag
	aliases  []string
	parents  []int
	children []int
	stashIDs []models.StashID
}

func loadTag(t *testing.T, r models.Repository, id int) *tagState {
	t.Helper()
	var ret *tagState
	require.NoError(t, r.WithReadTxn(context.Background(), func(ctx context.Context) error {
		tag, err := r.Tag.Find(ctx, id)
		if err != nil || tag == nil {
			return err
		}
		ret = &tagState{tag: tag}
		if ret.aliases, err = r.Tag.GetAliases(ctx, id); err != nil {
			return err
		}
		if ret.parents, err = r.Tag.GetParentIDs(ctx, id); err != nil {
			return err
		}
		if ret.children, err = r.Tag.GetChildIDs(ctx, id); err != nil {
			return err
		}
		ret.stashIDs, err = r.Tag.GetStashIDs(ctx, id)
		return err
	}))
	return ret
}

func sceneTagIDs(t *testing.T, r models.Repository, sceneID int) []int {
	t.Helper()
	var ret []int
	require.NoError(t, r.WithReadTxn(context.Background(), func(ctx context.Context) error {
		var err error
		ret, err = r.Scene.GetTagIDs(ctx, sceneID)
		return err
	}))
	return ret
}

func stashIDKeys(ids []models.StashID) []string {
	ret := make([]string, len(ids))
	for i, id := range ids {
		ret[i] = id.Endpoint + " " + id.StashID
	}
	return ret
}

func TestTranslateTag_RenameKeepsOldNameAsAlias(t *testing.T) {
	r := newTestRepository(t)
	parent := createTag(t, r, tagSpec{name: "Genre"})
	child := createTag(t, r, tagSpec{name: "Child"})
	id := createTag(t, r, tagSpec{
		name:        "中出し",
		description: "説明",
		aliases:     []string{"なかだし", "Creampie"},
		parents:     []int{parent},
		children:    []int{child},
		stashIDs:    []models.StashID{{Endpoint: endpointA, StashID: "a1"}},
	})
	sceneID := createScene(t, r, id)

	tr := &dictTranslator{dict: map[string]string{"中出し": "Creampie"}}
	res, err := TranslateTag(context.Background(), r, tr, id, tagTestOptions)
	require.NoError(t, err)

	assert.Equal(t, TagRenamed, res.Action)
	assert.Equal(t, "ja", res.Language)
	assert.Equal(t, "Description: 説明", tr.texts[0].Context)
	assert.Equal(t, translate.FieldTag, tr.texts[0].Field)

	got := loadTag(t, r, id)
	assert.Equal(t, "Creampie", got.tag.Name)
	// the alias that equals the new name is dropped
	assert.ElementsMatch(t, []string{"なかだし", "中出し"}, got.aliases)
	assert.Equal(t, []int{parent}, got.parents)
	assert.Equal(t, []int{child}, got.children)
	assert.Equal(t, []string{endpointA + " a1"}, stashIDKeys(got.stashIDs))
	assert.Equal(t, []int{id}, sceneTagIDs(t, r, sceneID))
}

func TestTranslateTag_Skips(t *testing.T) {
	r := newTestRepository(t)
	latin := createTag(t, r, tagSpec{name: "Creampie"})
	same := createTag(t, r, tagSpec{name: "コスプレ"})
	leftover := createTag(t, r, tagSpec{name: "素人"})

	tr := &dictTranslator{dict: map[string]string{"コスプレ": "コスプレ", "素人": "Amateur 素人"}}
	for id, reason := range map[int]string{
		latin:    "not in a mapped source language",
		leftover: "still contains ja text",
	} {
		res, err := TranslateTag(context.Background(), r, tr, id, tagTestOptions)
		require.NoError(t, err)
		assert.Equal(t, TagSkipped, res.Action)
		assert.Contains(t, res.Reason, reason)
	}

	res, err := TranslateTag(context.Background(), r, tr, same, tagTestOptions)
	require.NoError(t, err)
	assert.Equal(t, TagSkipped, res.Action)
	assert.Equal(t, "translation equals the current name", res.Reason)

	assert.Equal(t, "素人", loadTag(t, r, leftover).tag.Name)
}

func TestTranslateTag_EmptyTranslationFails(t *testing.T) {
	r := newTestRepository(t)
	id := createTag(t, r, tagSpec{name: "素人"})

	_, err := TranslateTag(context.Background(), r, &dictTranslator{}, id, tagTestOptions)
	assert.Error(t, err)
	assert.Equal(t, "素人", loadTag(t, r, id).tag.Name)
}

func TestTranslateTag_CollisionWithNameMerges(t *testing.T) {
	r := newTestRepository(t)
	dst := createTag(t, r, tagSpec{name: "Creampie", aliases: []string{"CP"}})
	src := createTag(t, r, tagSpec{name: "中出し", aliases: []string{"なかだし"}})
	srcScene := createScene(t, r, src)
	bothScene := createScene(t, r, src, dst)

	res, err := TranslateTag(context.Background(), r, &dictTranslator{dict: map[string]string{"中出し": "creampie"}}, src, tagTestOptions)
	require.NoError(t, err)

	require.Equal(t, TagMerged, res.Action)
	assert.Equal(t, dst, res.Merge.DestinationID)
	assert.Nil(t, loadTag(t, r, src), "source is deleted")

	got := loadTag(t, r, dst)
	assert.Equal(t, "Creampie", got.tag.Name)
	assert.ElementsMatch(t, []string{"CP", "中出し", "なかだし"}, got.aliases)
	assert.Equal(t, []int{dst}, sceneTagIDs(t, r, srcScene))
	assert.Equal(t, []int{dst}, sceneTagIDs(t, r, bothScene))
}

func TestTranslateTag_CollisionWithAliasMerges(t *testing.T) {
	r := newTestRepository(t)
	dst := createTag(t, r, tagSpec{name: "Cream Pie", aliases: []string{"CREAMPIE"}})
	src := createTag(t, r, tagSpec{name: "中出し"})

	res, err := TranslateTag(context.Background(), r, &dictTranslator{dict: map[string]string{"中出し": "Creampie"}}, src, tagTestOptions)
	require.NoError(t, err)
	require.Equal(t, TagMerged, res.Action)
	assert.Equal(t, dst, res.Merge.DestinationID)
	assert.Equal(t, "Cream Pie", loadTag(t, r, dst).tag.Name)
}

func TestTranslateTag_CollisionWithoutMergeSkips(t *testing.T) {
	r := newTestRepository(t)
	dst := createTag(t, r, tagSpec{name: "Creampie"})
	src := createTag(t, r, tagSpec{name: "中出し"})

	opts := tagTestOptions
	opts.MergeOnCollision = false
	res, err := TranslateTag(context.Background(), r, &dictTranslator{dict: map[string]string{"中出し": "Creampie"}}, src, opts)
	require.NoError(t, err)

	assert.Equal(t, TagSkipped, res.Action)
	assert.True(t, res.Collision)
	assert.Contains(t, res.Reason, `collides with tag "Creampie"`)
	assert.Equal(t, "中出し", loadTag(t, r, src).tag.Name)
	assert.Equal(t, "Creampie", loadTag(t, r, dst).tag.Name)
}

func TestMergeTag_StashIDs(t *testing.T) {
	r := newTestRepository(t)
	dst := createTag(t, r, tagSpec{name: "Creampie", stashIDs: []models.StashID{
		{Endpoint: endpointA, StashID: "same"},
	}})
	src := createTag(t, r, tagSpec{name: "中出し", stashIDs: []models.StashID{
		{Endpoint: endpointA, StashID: "same"},
		{Endpoint: endpointA, StashID: "different"},
		{Endpoint: endpointB, StashID: "b1"},
	}})

	var res *TagMergeResult
	require.NoError(t, r.WithTxn(context.Background(), func(ctx context.Context) error {
		var err error
		res, err = MergeTag(ctx, r.Tag, src, dst)
		return err
	}))

	assert.ElementsMatch(t, []string{endpointA + " same", endpointA + " different", endpointB + " b1"}, stashIDKeys(loadTag(t, r, dst).stashIDs))
	assert.Equal(t, []string{endpointA + " same"}, stashIDKeys(res.DuplicateStashIDs))
	assert.ElementsMatch(t, []string{endpointA + " different", endpointB + " b1"}, stashIDKeys(res.MovedStashIDs))
	require.Len(t, res.Warnings, 1)
	assert.Contains(t, res.Warnings[0], "several stash IDs for "+endpointA)
}

func mergeTags(t *testing.T, r models.Repository, src, dst int) *TagMergeResult {
	t.Helper()
	var res *TagMergeResult
	require.NoError(t, r.WithTxn(context.Background(), func(ctx context.Context) error {
		var err error
		res, err = MergeTag(ctx, r.Tag, src, dst)
		return err
	}))
	return res
}

func TestMergeTag_ParentsAreUnitedCycleSafe(t *testing.T) {
	r := newTestRepository(t)

	// grand -> dst -> below
	grand := createTag(t, r, tagSpec{name: "Grand"})
	below := createTag(t, r, tagSpec{name: "Below"})
	dst := createTag(t, r, tagSpec{name: "Creampie", parents: []int{grand}, children: []int{below}})
	newParent := createTag(t, r, tagSpec{name: "New parent"})
	newChild := createTag(t, r, tagSpec{name: "New child"})
	src := createTag(t, r, tagSpec{
		name: "中出し",
		// below is a descendant of dst, dst itself would be a self-reference,
		// grand is a parent already
		parents:  []int{newParent, below, grand, dst},
		children: []int{newChild},
	})

	res := mergeTags(t, r, src, dst)

	got := loadTag(t, r, dst)
	assert.ElementsMatch(t, []int{grand, newParent}, got.parents)
	assert.ElementsMatch(t, []int{below, newChild}, got.children)
	assert.Equal(t, []int{newParent}, res.AddedParentIDs)
	assert.Equal(t, []int{newChild}, res.AddedChildIDs)
	require.Len(t, res.SkippedRelations, 2)
	assert.Contains(t, res.String(), `parent "Below" (`+itoa(below)+`): would create a cycle`)
	assert.Contains(t, res.String(), `parent "Creampie" (`+itoa(dst)+`): would be a self-reference`)
}

func TestMergeTag_ChildrenAreUnitedCycleSafe(t *testing.T) {
	r := newTestRepository(t)

	// ancestor -> grand -> dst -> below
	ancestor := createTag(t, r, tagSpec{name: "Ancestor"})
	grand := createTag(t, r, tagSpec{name: "Grand", parents: []int{ancestor}})
	below := createTag(t, r, tagSpec{name: "Below"})
	dst := createTag(t, r, tagSpec{name: "Creampie", parents: []int{grand}, children: []int{below}})
	newParent := createTag(t, r, tagSpec{name: "New parent"})
	newChild := createTag(t, r, tagSpec{name: "New child"})
	src := createTag(t, r, tagSpec{
		name:    "中出し",
		parents: []int{newParent},
		// ancestor is an ancestor of dst, dst itself would be a
		// self-reference, below is a child already
		children: []int{newChild, ancestor, dst, below},
	})

	res := mergeTags(t, r, src, dst)

	got := loadTag(t, r, dst)
	assert.ElementsMatch(t, []int{grand, newParent}, got.parents)
	assert.ElementsMatch(t, []int{below, newChild}, got.children)
	assert.Equal(t, []int{newParent}, res.AddedParentIDs)
	assert.Equal(t, []int{newChild}, res.AddedChildIDs)
	require.Len(t, res.SkippedRelations, 2)
	assert.Contains(t, res.String(), `child "Ancestor" (`+itoa(ancestor)+`): would create a cycle`)
	assert.Contains(t, res.String(), `child "Creampie" (`+itoa(dst)+`): would be a self-reference`)

	// the destination is not its own ancestor
	require.NoError(t, r.WithReadTxn(context.Background(), func(ctx context.Context) error {
		descendants, err := r.Tag.FindAllDescendants(ctx, dst, nil)
		for _, d := range descendants {
			assert.NotContains(t, []int{ancestor, grand}, d.ID)
		}
		return err
	}))
}

func TestJob_TagsRenameThenMergeInSameRun(t *testing.T) {
	r := newTestRepository(t)
	first := createTag(t, r, tagSpec{name: "中出し"})
	second := createTag(t, r, tagSpec{name: "膣内射精"})
	russian := createTag(t, r, tagSpec{name: "Кремпай"})
	createTag(t, r, tagSpec{name: "Already English"})
	sceneID := createScene(t, r, second)

	hooks := &recordingTagHooks{}
	j := &Job{
		Repository: r,
		Translator: &dictTranslator{dict: map[string]string{"中出し": "Creampie", "膣内射精": "Creampie", "Кремпай": "Creampie (RU)"}},
		TagHooks:   hooks,
		Options:    tagTestOptions,
		Tags:       true,
	}

	res, err := j.Execute(context.Background(), noopProgress{})
	require.NoError(t, err)
	assert.Nil(t, res.Scenes)
	assert.Equal(t, TagSummary{Renamed: 2, Merged: 1}, *res.Tags)

	assert.Equal(t, "Creampie", loadTag(t, r, first).tag.Name)
	assert.ElementsMatch(t, []string{"中出し", "膣内射精"}, loadTag(t, r, first).aliases)
	assert.Nil(t, loadTag(t, r, second))
	assert.Equal(t, []int{first}, sceneTagIDs(t, r, sceneID))
	assert.Equal(t, "Creampie (RU)", loadTag(t, r, russian).tag.Name)

	require.Len(t, hooks.calls, 3)
	assert.Equal(t, hook.TagUpdatePost, hooks.calls[0].hookType)
	assert.Equal(t, hookCall{id: first, hookType: hook.TagMergePost, input: map[string]interface{}{
		"source": []string{itoa(second)}, "destination": itoa(first),
	}}, hooks.calls[1])
}

func TestJob_TagsAbortAfterConsecutiveFailures(t *testing.T) {
	r := newTestRepository(t)
	for _, name := range []string{"一", "二", "三", "四"} {
		createTag(t, r, tagSpec{name: name})
	}

	j := &Job{Repository: r, Translator: &dictTranslator{}, Options: tagTestOptions, Tags: true}
	res, err := j.Execute(context.Background(), noopProgress{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "3 consecutive failures")
	assert.Equal(t, TagSummary{Failed: 3}, *res.Tags)
}

func itoa(i int) string { return strconv.Itoa(i) }

func TestMain(m *testing.M) {
	// initialise empty config - needed by some db migrations
	_ = config.InitializeEmpty()
	os.Exit(m.Run())
}
