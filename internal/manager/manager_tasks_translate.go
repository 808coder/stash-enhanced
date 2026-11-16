package manager

import (
	"context"

	"github.com/stashapp/stash/internal/translation"
	"github.com/stashapp/stash/pkg/job"
)

// TranslateMetadataOptions selects what a translation job translates.
type TranslateMetadataOptions struct {
	// Scenes translates scene titles and details.
	Scenes bool
	// SceneIDs limits the scenes. If empty, all scenes with a mapped language
	// tag are translated.
	SceneIDs []int
	// Force re-translates scene fields from their original metadata.
	Force bool
	// Tags translates tag names.
	Tags bool
	// MergeOnCollision merges a tag into the tag whose name or alias matches
	// its translation.
	MergeOnCollision bool
}

// TranslateMetadata queues a job that translates and saves scene metadata
// and tag names. It fails right away if no translation provider is
// configured.
func (s *Manager) TranslateMetadata(ctx context.Context, opts TranslateMetadataOptions) (int, error) {
	tr, err := translation.NewTranslator(s.Config)
	if err != nil {
		return 0, err
	}

	t := &translation.Job{
		Repository: s.Repository,
		Translator: tr,
		SceneHooks: s.PluginCache,
		TagHooks:   s.PluginCache,
		Options:    translation.OptionsFromConfig(s.Config, opts.Force, opts.MergeOnCollision),
		Scenes:     opts.Scenes,
		SceneIDs:   opts.SceneIDs,
		Tags:       opts.Tags,
	}

	j := job.MakeJobExec(func(ctx context.Context, progress *job.Progress) error {
		_, err := t.Execute(ctx, progress)
		return err
	})

	return s.JobManager.Add(ctx, "Translating metadata...", j), nil
}
