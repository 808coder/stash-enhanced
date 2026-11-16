package api

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/internal/translation"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/plugin/hook"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
)

// sceneHooks adapts the resolver's hook executor to translation.SceneHookExecutor.
type sceneHooks struct {
	r *mutationResolver
}

func (h sceneHooks) ExecuteSceneUpdatePostHooks(ctx context.Context, input models.SceneUpdateInput, fields []string) {
	id, err := strconv.Atoi(input.ID)
	if err != nil {
		return
	}
	h.r.hookExecutor.ExecutePostHooks(ctx, id, hook.SceneUpdatePost, input, fields)
}

func toFieldTranslationResult(f *translation.FieldTranslation) *SceneFieldTranslation {
	if f == nil {
		return nil
	}
	return &SceneFieldTranslation{Source: f.Source, Current: f.Current, Translation: f.Translation, Language: f.Language}
}

func (r *mutationResolver) makeSceneTranslation(ctx context.Context, t *translation.SceneTranslation, saved bool) (*SceneTranslation, error) {
	var s *models.Scene
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		var err error
		s, err = r.repository.Scene.Find(ctx, t.SceneID)
		return err
	}); err != nil {
		return nil, err
	}

	warnings := t.Warnings
	if warnings == nil {
		warnings = []string{}
	}

	return &SceneTranslation{
		Scene:    s,
		Provider: t.Provider,
		Title:    toFieldTranslationResult(t.Title),
		Details:  toFieldTranslationResult(t.Details),
		Warnings: warnings,
		Saved:    saved,
	}, nil
}

func (r *mutationResolver) translateScene(ctx context.Context, id string, force *bool, save bool) (*SceneTranslation, error) {
	sceneID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("converting id: %w", err)
	}

	c := config.GetInstance()
	tr, err := translation.NewTranslator(c)
	if err != nil {
		return nil, err
	}
	opts := translation.OptionsFromConfig(c, force != nil && *force, false)

	var t *translation.SceneTranslation
	if save {
		t, err = translation.TranslateAndApply(ctx, r.repository, tr, sceneHooks{r}, sceneID, opts, time.Now())
	} else {
		t, err = translation.Translate(ctx, r.repository, tr, sceneID, opts)
	}
	if err != nil {
		return nil, err
	}

	return r.makeSceneTranslation(ctx, t, save && !t.Empty())
}

func (r *mutationResolver) SceneTranslatePreview(ctx context.Context, id string, force *bool) (*SceneTranslation, error) {
	return r.translateScene(ctx, id, force, false)
}

func (r *mutationResolver) SceneTranslate(ctx context.Context, id string, force *bool) (*SceneTranslation, error) {
	return r.translateScene(ctx, id, force, true)
}

func fromFieldTranslationInput(f *SceneFieldTranslationInput) *translation.FieldTranslation {
	if f == nil {
		return nil
	}
	return &translation.FieldTranslation{Source: f.Source, Translation: strings.TrimSpace(f.Translation), Language: f.Language}
}

func (r *mutationResolver) SceneTranslateApply(ctx context.Context, input SceneTranslationApplyInput) (*models.Scene, error) {
	sceneID, err := strconv.Atoi(input.ID)
	if err != nil {
		return nil, fmt.Errorf("converting id: %w", err)
	}

	t := &translation.SceneTranslation{
		SceneID:  sceneID,
		Provider: input.Provider,
		Title:    fromFieldTranslationInput(input.Title),
		Details:  fromFieldTranslationInput(input.Details),
	}
	if t.Empty() {
		return nil, errors.New("nothing to apply")
	}

	var hookInput *models.SceneUpdateInput
	var fields []string
	var ret *models.Scene
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		var err error
		hookInput, fields, err = translation.Apply(ctx, r.repository, t, time.Now())
		if err != nil {
			return err
		}
		ret, err = r.repository.Scene.Find(ctx, sceneID)
		return err
	}); err != nil {
		return nil, err
	}

	translation.LogApplied(t)
	sceneHooks{r}.ExecuteSceneUpdatePostHooks(ctx, *hookInput, fields)
	return ret, nil
}

func (r *mutationResolver) ScenesRestoreOriginal(ctx context.Context, ids []string) ([]*models.Scene, error) {
	sceneIDs, err := stringslice.StringSliceToIntSlice(ids)
	if err != nil {
		return nil, fmt.Errorf("converting ids: %w", err)
	}

	type restored struct {
		sceneID int
		input   *models.SceneUpdateInput
		fields  []string
	}
	var hooks []restored
	var ret []*models.Scene

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		for _, id := range sceneIDs {
			input, fields, err := translation.Restore(ctx, r.repository, id)
			if err != nil {
				return fmt.Errorf("restoring scene %d: %w", id, err)
			}
			if input != nil {
				hooks = append(hooks, restored{id, input, fields})
			}
		}

		var err error
		ret, err = r.repository.Scene.FindMany(ctx, sceneIDs)
		return err
	}); err != nil {
		return nil, err
	}

	for _, h := range hooks {
		translation.LogRestored(h.sceneID)
		sceneHooks{r}.ExecuteSceneUpdatePostHooks(ctx, *h.input, h.fields)
	}
	return ret, nil
}

func (r *mutationResolver) MetadataTranslate(ctx context.Context, input TranslateMetadataInput) (string, error) {
	sceneIDs, err := stringslice.StringSliceToIntSlice(input.SceneIds)
	if err != nil {
		return "", fmt.Errorf("converting ids: %w", err)
	}

	boolOr := func(b *bool, def bool) bool {
		if b == nil {
			return def
		}
		return *b
	}

	jobID, err := manager.GetInstance().TranslateMetadata(ctx, manager.TranslateMetadataOptions{
		Scenes:           boolOr(input.Scenes, true),
		SceneIDs:         sceneIDs,
		Force:            boolOr(input.Force, false),
		Tags:             boolOr(input.Tags, false),
		MergeOnCollision: boolOr(input.MergeOnCollision, true),
	})
	if err != nil {
		return "", err
	}
	return strconv.Itoa(jobID), nil
}
