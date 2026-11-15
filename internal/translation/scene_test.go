package translation

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/models/mocks"
	"github.com/stashapp/stash/pkg/translate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	tagJapanese = 10
	tagRussian  = 11
	tagOther    = 12
)

var testOptions = Options{
	LanguageMappings: map[int]string{tagJapanese: "ja", tagRussian: "ru"},
	TargetLanguage:   "en",
}

func withForce(o Options) Options {
	o.Force = true
	return o
}

func TestPlanScene(t *testing.T) {
	tests := []struct {
		name         string
		data         sceneData
		opts         Options
		wantTitle    *fieldPlan
		wantDetails  *fieldPlan
		wantWarnings int
	}{
		{
			name: "no language tag",
			data: sceneData{title: "ある夏の日", tagIDs: []int{tagOther}},
			opts: testOptions,
		},
		{
			name:        "japanese title and details",
			data:        sceneData{title: "ある夏の日", details: "夏の話", tagIDs: []int{tagJapanese, tagOther}},
			opts:        testOptions,
			wantTitle:   &fieldPlan{source: "ある夏の日", current: "ある夏の日", language: "ja"},
			wantDetails: &fieldPlan{source: "夏の話", current: "夏の話", language: "ja"},
		},
		{
			name:        "each field on its own: title already english",
			data:        sceneData{title: "A Summer Day", details: "夏の話", tagIDs: []int{tagJapanese}},
			opts:        testOptions,
			wantDetails: &fieldPlan{source: "夏の話", current: "夏の話", language: "ja"},
		},
		{
			name: "already translated, no force",
			data: sceneData{title: "A Summer Day", tagIDs: []int{tagJapanese}, customFields: map[string]interface{}{
				models.CustomFieldOriginalTitle: "ある夏の日",
			}},
			opts: testOptions,
		},
		{
			name: "force translates from the original",
			data: sceneData{title: "A Summer Day", tagIDs: []int{tagJapanese}, customFields: map[string]interface{}{
				models.CustomFieldOriginalTitle: "ある夏の日",
				CustomFieldSourceLanguage:       "ja",
			}},
			opts:      withForce(testOptions),
			wantTitle: &fieldPlan{source: "ある夏の日", current: "A Summer Day", language: "ja"},
		},
		{
			name:      "force without stored language detects it",
			data:      sceneData{title: "A Summer Day", tagIDs: []int{tagJapanese}, customFields: map[string]interface{}{models.CustomFieldOriginalTitle: "ある夏の日"}},
			opts:      withForce(testOptions),
			wantTitle: &fieldPlan{source: "ある夏の日", current: "A Summer Day", language: "ja"},
		},
		{
			name: "force without original skips",
			data: sceneData{title: "A Summer Day", tagIDs: []int{tagJapanese}},
			opts: withForce(testOptions),
		},
		{
			name: "source text written again becomes the new original",
			data: sceneData{title: "新しい夏の日", tagIDs: []int{tagJapanese}, customFields: map[string]interface{}{
				models.CustomFieldOriginalTitle: "ある夏の日",
			}},
			opts:      testOptions,
			wantTitle: &fieldPlan{source: "新しい夏の日", current: "新しい夏の日", language: "ja"},
		},
		{
			name: "leftover source text in a translation keeps the original",
			data: sceneData{title: "A温泉 trip with my mother", tagIDs: []int{tagJapanese}, customFields: map[string]interface{}{
				models.CustomFieldOriginalTitle: "母と温泉旅行",
			}},
			opts:         testOptions,
			wantWarnings: 1,
		},
		{
			name: "leftover source text with force translates the original",
			data: sceneData{title: "A温泉 trip with my mother", tagIDs: []int{tagJapanese}, customFields: map[string]interface{}{
				models.CustomFieldOriginalTitle: "母と温泉旅行",
			}},
			opts:      withForce(testOptions),
			wantTitle: &fieldPlan{source: "母と温泉旅行", current: "A温泉 trip with my mother", language: "ja"},
		},
		{
			name:      "leftover source text without stored original is translated",
			data:      sceneData{title: "A温泉 trip", tagIDs: []int{tagJapanese}},
			opts:      testOptions,
			wantTitle: &fieldPlan{source: "A温泉 trip", current: "A温泉 trip", language: "ja"},
		},
		{
			name:         "two language tags: script decides, with warning",
			data:         sceneData{title: "Летний день", tagIDs: []int{tagJapanese, tagRussian}},
			opts:         testOptions,
			wantTitle:    &fieldPlan{source: "Летний день", current: "Летний день", language: "ru"},
			wantWarnings: 1,
		},
		{
			name: "ambiguous script is skipped with warning",
			data: sceneData{title: "巨乳", tagIDs: []int{1, 2}},
			opts: Options{
				LanguageMappings: map[int]string{1: "ja", 2: "zh"},
				TargetLanguage:   "en",
			},
			wantWarnings: 2,
		},
		{
			name: "latin-script language is never detected",
			data: sceneData{title: "Un día de verano", tagIDs: []int{1}},
			opts: Options{LanguageMappings: map[int]string{1: "es"}, TargetLanguage: "en"},
		},
	}

	for _, tt := range tests {
		got := planScene(tt.data, tt.opts)
		assert.Equal(t, tt.wantTitle, got.title, tt.name+": title")
		assert.Equal(t, tt.wantDetails, got.details, tt.name+": details")
		assert.Len(t, got.warnings, tt.wantWarnings, tt.name+": warnings %v", got.warnings)
	}
}

func TestTitleContext(t *testing.T) {
	d := sceneData{
		title:   "ある夏の日",
		details: "夏の話",
		performers: []performerName{
			{name: "Yua Mikami", aliases: []string{"三上悠亜"}},
			{name: "Jane Doe"},
		},
	}

	p := planScene(sceneData{title: d.title, details: d.details, tagIDs: []int{tagJapanese}}, testOptions)
	assert.Equal(t, "Performers: Yua Mikami (also: 三上悠亜); Jane Doe\nDetails: 夏の話", titleContext(d, p))

	// details already translated: the original is the context
	d2 := sceneData{title: "ある夏の日", details: "A summer story", customFields: map[string]interface{}{models.CustomFieldOriginalDetails: "夏の話"}}
	assert.Equal(t, "Details: 夏の話", titleContext(d2, scenePlan{}))

	assert.Equal(t, "", titleContext(sceneData{}, scenePlan{}))
}

// fakeTranslator prefixes texts with "EN:" and records calls.
type fakeTranslator struct {
	texts []translate.Text
	err   error
}

func (f *fakeTranslator) Name() string { return "fake:model" }

func (f *fakeTranslator) Translate(_ context.Context, texts []translate.Text, from, to string) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	ret := make([]string, len(texts))
	for i, t := range texts {
		f.texts = append(f.texts, t)
		ret[i] = "EN(" + from + ">" + to + "):" + t.Value
	}
	return ret, nil
}

func mockScene(db *mocks.Database, id int, title, details string, tagIDs, performerIDs []int, cf map[string]interface{}) {
	db.Scene.On("Find", mock.Anything, id).Return(&models.Scene{ID: id, Title: title, Details: details}, nil)
	db.Scene.On("GetTagIDs", mock.Anything, id).Return(tagIDs, nil)
	db.Scene.On("GetPerformerIDs", mock.Anything, id).Return(performerIDs, nil)
	db.Scene.On("GetCustomFields", mock.Anything, id).Return(cf, nil)
}

func TestTranslate(t *testing.T) {
	db := mocks.NewDatabase()
	mockScene(db, 1, "ある夏の日", "夏の話", []int{tagJapanese}, []int{5}, map[string]interface{}{})
	db.Performer.On("FindMany", mock.Anything, []int{5}).Return([]*models.Performer{{ID: 5, Name: "Yua Mikami"}}, nil)
	db.Performer.On("GetAliases", mock.Anything, 5).Return([]string{"三上悠亜"}, nil)

	tr := &fakeTranslator{}
	got, err := Translate(context.Background(), db.Repository(), tr, 1, testOptions)
	require.NoError(t, err)

	assert.Equal(t, "fake:model", got.Provider)
	assert.Equal(t, &FieldTranslation{Source: "ある夏の日", Current: "ある夏の日", Translation: "EN(ja>en):ある夏の日", Language: "ja"}, got.Title)
	assert.Equal(t, "EN(ja>en):夏の話", got.Details.Translation)
	assert.Equal(t, "ja", got.SourceLanguage())

	require.Len(t, tr.texts, 2)
	assert.Equal(t, translate.FieldTitle, tr.texts[0].Field)
	assert.Equal(t, "Performers: Yua Mikami (also: 三上悠亜)\nDetails: 夏の話", tr.texts[0].Context)
	assert.Equal(t, translate.FieldDetails, tr.texts[1].Field)
	assert.Empty(t, tr.texts[1].Context, "details get no context")
}

func TestApply(t *testing.T) {
	db := mocks.NewDatabase()
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)

	db.Scene.On("UpdatePartial", mock.Anything, 1, mock.MatchedBy(func(p models.ScenePartial) bool {
		return p.Title.Value == "A Summer Day" && p.Title.Set && !p.Details.Set
	})).Return(&models.Scene{ID: 1}, nil).Once()
	db.Scene.On("SetCustomFields", mock.Anything, 1, models.CustomFieldsInput{Partial: map[string]interface{}{
		models.CustomFieldOriginalTitle: "ある夏の日",
		CustomFieldSourceLanguage:       "ja",
		CustomFieldProvider:             "fake:model",
		CustomFieldTranslatedAt:         "2026-11-15T12:00:00Z",
	}}).Return(nil).Once()

	hookInput, fields, err := Apply(context.Background(), db.Repository(), &SceneTranslation{
		SceneID:  1,
		Provider: "fake:model",
		Title:    &FieldTranslation{Source: "ある夏の日", Translation: "A Summer Day", Language: "ja"},
	}, now)

	require.NoError(t, err)
	assert.Equal(t, "1", hookInput.ID)
	assert.Equal(t, "A Summer Day", *hookInput.Title)
	assert.Nil(t, hookInput.Details)
	assert.ElementsMatch(t, []string{"title", "custom_fields"}, fields)
	db.AssertExpectations(t)

	_, _, err = Apply(context.Background(), db.Repository(), &SceneTranslation{SceneID: 1}, now)
	assert.Error(t, err, "empty translation")

	_, _, err = Apply(context.Background(), db.Repository(), &SceneTranslation{SceneID: 1, Title: &FieldTranslation{Translation: " "}}, now)
	assert.Error(t, err, "blank title")
}

func TestRestore(t *testing.T) {
	db := mocks.NewDatabase()
	db.Scene.On("GetCustomFields", mock.Anything, 1).Return(map[string]interface{}{
		models.CustomFieldOriginalTitle: "ある夏の日",
		CustomFieldSourceLanguage:       "ja",
		CustomFieldProvider:             "fake:model",
		CustomFieldTranslatedAt:         "2026-11-15T12:00:00Z",
		"other":                         "kept",
	}, nil)
	db.Scene.On("UpdatePartial", mock.Anything, 1, mock.MatchedBy(func(p models.ScenePartial) bool {
		return p.Title.Value == "ある夏の日" && !p.Details.Set
	})).Return(&models.Scene{ID: 1}, nil).Once()
	db.Scene.On("SetCustomFields", mock.Anything, 1, mock.MatchedBy(func(in models.CustomFieldsInput) bool {
		return in.Partial == nil && in.Full == nil && len(in.Remove) == 4 && !strings.Contains(strings.Join(in.Remove, ","), "other")
	})).Return(nil).Once()

	hookInput, fields, err := Restore(context.Background(), db.Repository(), 1)
	require.NoError(t, err)
	assert.Equal(t, "ある夏の日", *hookInput.Title)
	assert.ElementsMatch(t, []string{"title", "custom_fields"}, fields)
	db.AssertExpectations(t)

	// nothing to restore
	db.Scene.On("GetCustomFields", mock.Anything, 2).Return(map[string]interface{}{"other": "x"}, nil)
	hookInput, _, err = Restore(context.Background(), db.Repository(), 2)
	require.NoError(t, err)
	assert.Nil(t, hookInput)
}

type noopProgress struct{}

func (noopProgress) SetTotal(int)                    {}
func (noopProgress) Increment()                      {}
func (noopProgress) ExecuteTask(_ string, fn func()) { fn() }

type recordingHooks struct{ calls int }

func (h *recordingHooks) ExecuteSceneUpdatePostHooks(context.Context, models.SceneUpdateInput, []string) {
	h.calls++
}

func TestJob_AbortsAfterConsecutiveFailures(t *testing.T) {
	db := mocks.NewDatabase()
	for id := 1; id <= 5; id++ {
		mockScene(db, id, "ある夏の日", "", []int{tagJapanese}, nil, map[string]interface{}{})
	}
	db.Performer.On("FindMany", mock.Anything, mock.Anything).Return(nil, nil)

	j := &Job{
		Repository: db.Repository(),
		Translator: &fakeTranslator{err: &translate.Error{Kind: translate.ErrorKindConnection, Err: errors.New("down")}},
		Options:    testOptions,
		SceneIDs:   []int{1, 2, 3, 4, 5},
	}

	summary, err := j.Execute(context.Background(), noopProgress{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "3 consecutive failures")
	assert.Equal(t, Summary{Failed: 3}, *summary)
}

func TestJob_TranslatesAndSkips(t *testing.T) {
	db := mocks.NewDatabase()
	mockScene(db, 1, "ある夏の日", "", []int{tagJapanese}, nil, map[string]interface{}{})
	mockScene(db, 2, "Already English", "", []int{tagJapanese}, nil, map[string]interface{}{})
	db.Performer.On("FindMany", mock.Anything, mock.Anything).Return(nil, nil)
	db.Scene.On("UpdatePartial", mock.Anything, 1, mock.Anything).Return(&models.Scene{ID: 1}, nil).Once()
	db.Scene.On("SetCustomFields", mock.Anything, 1, mock.Anything).Return(nil).Once()

	hooks := &recordingHooks{}
	j := &Job{
		Repository: db.Repository(),
		Translator: &fakeTranslator{},
		Hooks:      hooks,
		Options:    testOptions,
		SceneIDs:   []int{1, 2},
	}

	summary, err := j.Execute(context.Background(), noopProgress{})
	require.NoError(t, err)
	assert.Equal(t, Summary{Translated: 1, Skipped: 1}, *summary)
	assert.Equal(t, 1, hooks.calls)
	db.AssertExpectations(t)
}
