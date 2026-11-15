package translation

import (
	"context"
	"errors"
	"fmt"
	"strconv"
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

// HookExecutor runs Scene.Update.Post plugin hooks.
type HookExecutor interface {
	ExecuteSceneUpdatePostHooks(ctx context.Context, input models.SceneUpdateInput, inputFields []string)
}

// OptionsFromConfig reads the language mapping and target language from the
// configuration. Mappings with an invalid tag ID are ignored.
func OptionsFromConfig(c *config.Config, force bool) Options {
	opts := Options{
		LanguageMappings: make(map[int]string),
		TargetLanguage:   c.GetTranslationTargetLanguage(),
		Force:            force,
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
func TranslateAndApply(ctx context.Context, r models.Repository, tr translate.Translator, hooks HookExecutor, sceneID int, opts Options, now time.Time) (*SceneTranslation, error) {
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

	if hooks != nil {
		hooks.ExecuteSceneUpdatePostHooks(ctx, *hookInput, fields)
	}
	return t, nil
}

// Job translates scenes in a batch and saves the results directly.
type Job struct {
	Repository models.Repository
	Translator translate.Translator
	Hooks      HookExecutor
	Options    Options
	// SceneIDs to translate. If empty, all scenes with a mapped language tag
	// are translated.
	SceneIDs []int

	now func() time.Time
}

// Summary counts the outcome of a batch.
type Summary struct {
	Translated int
	Skipped    int
	Failed     int
}

func (s Summary) String() string {
	return fmt.Sprintf("%d translated, %d skipped, %d failed", s.Translated, s.Skipped, s.Failed)
}

func (j *Job) sceneIDs(ctx context.Context) ([]int, error) {
	if len(j.SceneIDs) > 0 {
		return j.SceneIDs, nil
	}

	if len(j.Options.LanguageMappings) == 0 {
		return nil, errors.New("no language mapping configured")
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

// Progress receives the progress of a batch. *job.Progress implements it.
type Progress interface {
	SetTotal(total int)
	Increment()
	ExecuteTask(description string, fn func())
}

// Execute runs the batch. It returns an error only if the batch could not
// start or was aborted after MaxConsecutiveFailures.
func (j *Job) Execute(ctx context.Context, progress Progress) (*Summary, error) {
	now := j.now
	if now == nil {
		now = time.Now
	}

	ids, err := j.sceneIDs(ctx)
	if err != nil {
		return nil, err
	}

	logger.Infof("Translating metadata of %d scenes with %s", len(ids), j.Translator.Name())
	progress.SetTotal(len(ids))

	summary := &Summary{}
	consecutiveFailures := 0

	for _, id := range ids {
		if job.IsCancelled(ctx) {
			logger.Info("Stopping metadata translation due to user request")
			break
		}

		var sceneErr error
		progress.ExecuteTask(fmt.Sprintf("Translating scene %d", id), func() {
			var t *SceneTranslation
			t, sceneErr = TranslateAndApply(ctx, j.Repository, j.Translator, j.Hooks, id, j.Options, now())
			if sceneErr != nil {
				return
			}
			for _, w := range t.Warnings {
				logger.Warnf("Translating scene %d: %s", id, w)
			}
			if t.Empty() {
				summary.Skipped++
				return
			}
			summary.Translated++
			logger.Infof("Translated scene %d from %s", id, t.SourceLanguage())
		})
		progress.Increment()

		if sceneErr != nil {
			if ctx.Err() != nil {
				break
			}
			summary.Failed++
			consecutiveFailures++
			logger.Errorf("Translating scene %d: %v", id, sceneErr)
			if consecutiveFailures >= MaxConsecutiveFailures {
				logger.Infof("Metadata translation: %s", summary)
				return summary, fmt.Errorf("aborting after %d consecutive failures: %w", consecutiveFailures, sceneErr)
			}
			continue
		}
		consecutiveFailures = 0
	}

	logger.Infof("Metadata translation: %s", summary)
	return summary, nil
}
