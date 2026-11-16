package scene

import (
	"context"
	"fmt"

	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
)

// applyTombstones checks whether the file matches a tombstone by oshash/md5.
// Manual tombstones tag the new scene and are kept; clean tombstones are
// removed without tagging. Must be called before the scene is created.
func (h *ScanHandler) applyTombstones(ctx context.Context, newScene *models.Scene, f *models.VideoFile) error {
	opts := h.Tombstones
	if opts == nil {
		return nil
	}

	oshash := f.Fingerprints.GetString(models.FingerprintTypeOshash)
	md5 := f.Fingerprints.GetString(models.FingerprintTypeMD5)

	tombstones, err := opts.Store.FindByFileFingerprints(ctx, oshash, md5)
	if err != nil {
		return fmt.Errorf("finding tombstones: %w", err)
	}

	manual := false
	for _, t := range tombstones {
		switch t.Origin {
		case models.TombstoneOriginManual:
			manual = true
		case models.TombstoneOriginClean:
			if err := opts.Store.Destroy(ctx, t.ID); err != nil {
				return fmt.Errorf("removing tombstone %d: %w", t.ID, err)
			}
			logger.Infof("Removed clean tombstone %d: %s was scanned again", t.ID, f.Base().Path)
		}
	}

	if !manual {
		return nil
	}

	tagID, err := opts.previouslyDeletedTagID(ctx)
	if err != nil {
		return err
	}

	logger.Infof("%s matches a manually deleted scene, adding tag", f.Base().Path)
	newScene.TagIDs = models.NewRelatedIDs([]int{tagID})

	return nil
}

func (o *ScanTombstoneOptions) previouslyDeletedTagID(ctx context.Context) (int, error) {
	if o.PreviouslyDeletedTagID != 0 {
		t, err := o.Tags.Find(ctx, o.PreviouslyDeletedTagID)
		if err != nil {
			return 0, fmt.Errorf("finding previously deleted tag: %w", err)
		}
		if t != nil {
			return t.ID, nil
		}
		logger.Warnf("Previously deleted tag %d no longer exists, using %q", o.PreviouslyDeletedTagID, fallbackPreviouslyDeletedTagName)
	}

	t, err := o.Tags.FindByName(ctx, fallbackPreviouslyDeletedTagName, true)
	if err != nil {
		return 0, fmt.Errorf("finding previously deleted tag: %w", err)
	}
	if t != nil {
		return t.ID, nil
	}

	newTag := models.NewTag()
	newTag.Name = fallbackPreviouslyDeletedTagName
	if err := o.Tags.Create(ctx, &models.CreateTagInput{Tag: &newTag}); err != nil {
		return 0, fmt.Errorf("creating previously deleted tag: %w", err)
	}

	return newTag.ID, nil
}
