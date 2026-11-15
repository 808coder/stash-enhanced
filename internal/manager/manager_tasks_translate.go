package manager

import (
	"context"

	"github.com/stashapp/stash/internal/translation"
	"github.com/stashapp/stash/pkg/job"
)

// TranslateMetadata queues a job that translates and saves scene metadata.
// It fails right away if no translation provider is configured.
func (s *Manager) TranslateMetadata(ctx context.Context, sceneIDs []int, force bool) (int, error) {
	tr, err := translation.NewTranslator(s.Config)
	if err != nil {
		return 0, err
	}

	t := &translation.Job{
		Repository: s.Repository,
		Translator: tr,
		Hooks:      s.PluginCache,
		Options:    translation.OptionsFromConfig(s.Config, force),
		SceneIDs:   sceneIDs,
	}

	j := job.MakeJobExec(func(ctx context.Context, progress *job.Progress) error {
		_, err := t.Execute(ctx, progress)
		return err
	})

	return s.JobManager.Add(ctx, "Translating metadata...", j), nil
}
