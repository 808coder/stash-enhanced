package translation

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/job"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/scene"
	"github.com/stashapp/stash/pkg/translate"
)

// MaxConsecutiveFailures aborts a batch after this many failed scenes in a
// row, e.g. because the provider is down or the API key is wrong.
const MaxConsecutiveFailures = 3

// SceneHookExecutor runs Scene.Update.Post plugin hooks.
type SceneHookExecutor interface {
	ExecuteSceneUpdatePostHooks(ctx context.Context, input models.SceneUpdateInput, inputFields []string)
}

// OptionsFromConfig reads the language mapping and target language from the
// configuration. Mappings with an invalid tag ID are ignored.
func OptionsFromConfig(c *config.Config, force, mergeOnCollision bool) Options {
	opts := Options{
		LanguageMappings: make(map[int]string),
		TargetLanguage:   c.GetTranslationTargetLanguage(),
		Force:            force,
		MergeOnCollision: mergeOnCollision,
	}
	for _, m := range c.GetTranslationLanguageMappings() {
		id, err := strconv.Atoi(m.TagID)
		if err != nil {
			logger.Warnf("translation: ignoring language mapping with invalid tag ID %q", m.TagID)
			continue
		}
		opts.LanguageMappings[id] = m.LanguageCode
	}
	return opts
}

// NewTranslator returns the active translation provider.
func NewTranslator(c *config.Config) (translate.Translator, error) {
	return translate.New(c.GetTranslationConfig(), nil)
}

// TranslateAndApply translates a scene and saves the result. It returns the
// translation, which is empty if nothing needed translating.
func TranslateAndApply(ctx context.Context, r models.Repository, tr translate.Translator, hooks SceneHookExecutor, sceneID int, opts Options, now time.Time) (*SceneTranslation, error) {
	t, err := Translate(ctx, r, tr, sceneID, opts)
	if err != nil {
		return nil, err
	}
	if t.Empty() {
		return t, nil
	}

	var hookInput *models.SceneUpdateInput
	var fields []string
	if err := r.WithTxn(ctx, func(ctx context.Context) error {
		var err error
		hookInput, fields, err = Apply(ctx, r, t, now)
		return err
	}); err != nil {
		return nil, err
	}

	LogApplied(t)

	if hooks != nil {
		hooks.ExecuteSceneUpdatePostHooks(ctx, *hookInput, fields)
	}
	return t, nil
}

// Job translates scenes and tags in a batch and saves the results directly.
type Job struct {
	Repository models.Repository
	Translator translate.Translator
	SceneHooks SceneHookExecutor
	TagHooks   TagHookExecutor
	Options    Options

	// Scenes translates scene titles and details.
	Scenes bool
	// SceneIDs to translate. If empty, all scenes with a mapped language tag
	// are translated.
	SceneIDs []int
	// Tags translates the names of all tags in a mapped source language.
	Tags bool

	now func() time.Time
}

// Summary counts the outcome of the scenes in a batch.
type Summary struct {
	Translated int
	Skipped    int
	Failed     int
}

func (s Summary) String() string {
	return fmt.Sprintf("%d translated, %d skipped, %d failed", s.Translated, s.Skipped, s.Failed)
}

// TagSummary counts the outcome of the tags in a batch.
type TagSummary struct {
	Renamed int
	Merged  int
	Skipped int
	Failed  int
}

func (s TagSummary) String() string {
	return fmt.Sprintf("%d renamed, %d merged, %d skipped, %d failed", s.Renamed, s.Merged, s.Skipped, s.Failed)
}

// Result is the outcome of a batch. A summary is nil if the batch did not
// include scenes or tags.
type Result struct {
	Scenes *Summary
	Tags   *TagSummary
}

func (r Result) String() string {
	var parts []string
	if r.Scenes != nil {
		parts = append(parts, "scenes: "+r.Scenes.String())
	}
	if r.Tags != nil {
		parts = append(parts, "tags: "+r.Tags.String())
	}
	return strings.Join(parts, "; ")
}

func (j *Job) sceneIDs(ctx context.Context) ([]int, error) {
	if len(j.SceneIDs) > 0 {
		return j.SceneIDs, nil
	}

	tagIDs := make([]string, 0, len(j.Options.LanguageMappings))
	for id := range j.Options.LanguageMappings {
		tagIDs = append(tagIDs, strconv.Itoa(id))
	}

	filter := &models.SceneFilterType{
		Tags: &models.HierarchicalMultiCriterionInput{
			Value:    tagIDs,
			Modifier: models.CriterionModifierIncludes,
		},
	}

	var ret []int
	err := j.Repository.WithReadTxn(ctx, func(ctx context.Context) error {
		return scene.BatchProcess(ctx, j.Repository.Scene, filter, nil, func(s *models.Scene) error {
			ret = append(ret, s.ID)
			return nil
		})
	})
	return ret, err
}

// tagIDs returns the tags whose name is in a mapped source language, or
// whose source language is ambiguous, sorted by ID.
func (j *Job) tagIDs(ctx context.Context) ([]int, error) {
	langs := j.Options.mappedLanguages()

	var ret []int
	err := j.Repository.WithReadTxn(ctx, func(ctx context.Context) error {
		tags, err := j.Repository.Tag.All(ctx)
		if err != nil {
			return err
		}
		for _, t := range tags {
			lang, err := TagSourceLanguage(t.Name, langs, j.Options.TargetLanguage)
			if lang != "" || errors.Is(err, translate.ErrAmbiguousLanguage) {
				ret = append(ret, t.ID)
			}
		}
		return nil
	})
	sort.Ints(ret)
	return ret, err
}

// Progress receives the progress of a batch. *job.Progress implements it.
type Progress interface {
	SetTotal(total int)
	Increment()
	ExecuteTask(description string, fn func())
}

// Execute runs the batch. It returns an error only if the batch could not
// start or was aborted after MaxConsecutiveFailures.
func (j *Job) Execute(ctx context.Context, progress Progress) (*Result, error) {
	now := j.now
	if now == nil {
		now = time.Now
	}

	if !j.Scenes && !j.Tags {
		return nil, errors.New("nothing to translate: select scenes or tags")
	}
	if len(j.Options.LanguageMappings) == 0 && (j.Tags || len(j.SceneIDs) == 0) {
		return nil, errors.New("no language mapping configured")
	}

	res := &Result{}
	var sceneIDs, tagIDs []int
	var err error
	if j.Scenes {
		res.Scenes = &Summary{}
		if sceneIDs, err = j.sceneIDs(ctx); err != nil {
			return nil, err
		}
	}
	if j.Tags {
		res.Tags = &TagSummary{}
		if tagIDs, err = j.tagIDs(ctx); err != nil {
			return nil, err
		}
	}

	logger.Infof("Translating metadata of %d scenes and %d tags with %s", len(sceneIDs), len(tagIDs), j.Translator.Name())
	progress.SetTotal(len(sceneIDs) + len(tagIDs))

	consecutiveFailures := 0
	// run translates one item. It returns true and an error if the batch
	// must stop.
	run := func(description string, failed *int, fn func() error) (bool, error) {
		if job.IsCancelled(ctx) {
			logger.Info("Stopping metadata translation due to user request")
			return true, nil
		}

		var itemErr error
		progress.ExecuteTask(description, func() { itemErr = fn() })
		progress.Increment()

		if itemErr == nil {
			consecutiveFailures = 0
			return false, nil
		}
		if ctx.Err() != nil {
			return true, nil
		}
		*failed++
		consecutiveFailures++
		logger.Errorf("%s: %v", description, itemErr)
		if consecutiveFailures >= MaxConsecutiveFailures {
			return true, fmt.Errorf("aborting after %d consecutive failures: %w", consecutiveFailures, itemErr)
		}
		return false, nil
	}

	finish := func(err error) (*Result, error) {
		logger.Infof("Metadata translation: %s", res)
		return res, err
	}

	for _, id := range sceneIDs {
		stop, err := run(fmt.Sprintf("Translating scene %d", id), &res.Scenes.Failed, func() error {
			t, err := TranslateAndApply(ctx, j.Repository, j.Translator, j.SceneHooks, id, j.Options, now())
			if err != nil {
				return err
			}
			if t.Empty() {
				res.Scenes.Skipped++
			} else {
				res.Scenes.Translated++
			}
			return nil
		})
		if stop {
			return finish(err)
		}
	}

	for _, id := range tagIDs {
		stop, err := run(fmt.Sprintf("Translating tag %d", id), &res.Tags.Failed, func() error {
			t, err := TranslateTag(ctx, j.Repository, j.Translator, id, j.Options)
			if err != nil {
				return err
			}
			LogTagTranslation(t)
			ExecuteTagHooks(ctx, j.TagHooks, t)
			switch t.Action {
			case TagRenamed:
				res.Tags.Renamed++
			case TagMerged:
				res.Tags.Merged++
			default:
				res.Tags.Skipped++
			}
			return nil
		})
		if stop {
			return finish(err)
		}
	}

	return finish(nil)
}
