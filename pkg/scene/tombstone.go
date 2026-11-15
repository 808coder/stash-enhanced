package scene

import (
	"context"
	"fmt"

	"github.com/stashapp/stash/pkg/models"
)

// Custom field keys holding the Original metadata of a translated scene.
// See docs/adr/0001-original-metadata-in-custom-fields.md.
const (
	customFieldOriginalTitle   = "original_title"
	customFieldOriginalDetails = "original_details"
)

// TombstoneOptions requests a Tombstone when a scene is destroyed.
//
// A Tombstone is only written while the "Record tombstones" setting is on.
// A manual Tombstone is only written when the scene's files are deleted from
// disk (DestroyOptions.DeleteFile); a clean Tombstone is always written,
// since the Clean task only destroys scenes whose file is already gone.
type TombstoneOptions struct {
	Origin models.TombstoneOrigin
	// Reason is optional free text; empty means none.
	Reason string
}

func (s *Service) shouldRecordTombstone(opts DestroyOptions) bool {
	if opts.Tombstone == nil || !s.Config.GetTombstonesRecord() {
		return false
	}

	switch opts.Tombstone.Origin {
	case models.TombstoneOriginManual:
		return opts.DeleteFile
	case models.TombstoneOriginClean:
		return true
	default:
		return false
	}
}

// recordTombstone snapshots the scene and stores it as a Tombstone. It must
// run before any of the scene's rows are removed.
func (s *Service) recordTombstone(ctx context.Context, scene *models.Scene, opts TombstoneOptions) error {
	t, err := s.snapshot(ctx, scene)
	if err != nil {
		return err
	}

	t.Origin = opts.Origin
	t.Reason = opts.Reason

	return s.Tombstone.Create(ctx, t)
}

func (s *Service) snapshot(ctx context.Context, scene *models.Scene) (*models.Tombstone, error) {
	r := s.Repository

	if err := scene.LoadURLs(ctx, r); err != nil {
		return nil, fmt.Errorf("loading urls: %w", err)
	}
	if err := scene.LoadStashIDs(ctx, r); err != nil {
		return nil, fmt.Errorf("loading stash ids: %w", err)
	}
	if err := scene.LoadPerformerIDs(ctx, r); err != nil {
		return nil, fmt.Errorf("loading performer ids: %w", err)
	}
	if err := scene.LoadTagIDs(ctx, r); err != nil {
		return nil, fmt.Errorf("loading tag ids: %w", err)
	}
	if err := scene.LoadFiles(ctx, r); err != nil {
		return nil, fmt.Errorf("loading files: %w", err)
	}

	customFields, err := r.GetCustomFields(ctx, scene.ID)
	if err != nil {
		return nil, fmt.Errorf("loading custom fields: %w", err)
	}
	oCount, err := r.GetOCount(ctx, scene.ID)
	if err != nil {
		return nil, fmt.Errorf("loading o-counter: %w", err)
	}
	playCount, err := r.CountViews(ctx, scene.ID)
	if err != nil {
		return nil, fmt.Errorf("loading play count: %w", err)
	}

	t := &models.Tombstone{
		SceneID:         scene.ID,
		Title:           scene.Title,
		Code:            scene.Code,
		Date:            scene.Date,
		URLs:            scene.URLs.List(),
		StashIDs:        toTombstoneStashIDs(scene.StashIDs.List()),
		OriginalTitle:   customFieldString(customFields, customFieldOriginalTitle),
		OriginalDetails: customFieldString(customFields, customFieldOriginalDetails),
		Details:         scene.Details,
		Rating:          scene.Rating,
		OCounter:        oCount,
		PlayCount:       playCount,
		SceneCreatedAt:  scene.CreatedAt,
	}

	if t.Studio, err = s.snapshotStudio(ctx, scene.StudioID); err != nil {
		return nil, err
	}
	if t.Performers, err = s.snapshotPerformers(ctx, scene.PerformerIDs.List()); err != nil {
		return nil, err
	}
	if t.Tags, err = s.snapshotTags(ctx, scene.TagIDs.List()); err != nil {
		return nil, err
	}

	for _, f := range scene.Files.List() {
		t.Files = append(t.Files, toTombstoneFile(f))
	}

	return t, nil
}

func (s *Service) snapshotStudio(ctx context.Context, studioID *int) (*models.TombstoneEntity, error) {
	if studioID == nil {
		return nil, nil
	}

	studio, err := s.Studio.Find(ctx, *studioID)
	if err != nil {
		return nil, fmt.Errorf("finding studio %d: %w", *studioID, err)
	}
	if studio == nil {
		return nil, nil
	}

	if err := studio.LoadStashIDs(ctx, s.Studio); err != nil {
		return nil, fmt.Errorf("loading studio stash ids: %w", err)
	}

	return &models.TombstoneEntity{
		Name:     studio.Name,
		StashIDs: toTombstoneStashIDs(studio.StashIDs.List()),
	}, nil
}

func (s *Service) snapshotPerformers(ctx context.Context, ids []int) ([]models.TombstoneEntity, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	performers, err := s.Performer.FindMany(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("finding performers: %w", err)
	}

	var ret []models.TombstoneEntity
	for _, p := range performers {
		if p == nil {
			continue
		}
		if err := p.LoadStashIDs(ctx, s.Performer); err != nil {
			return nil, fmt.Errorf("loading performer stash ids: %w", err)
		}
		ret = append(ret, models.TombstoneEntity{
			Name:     p.Name,
			StashIDs: toTombstoneStashIDs(p.StashIDs.List()),
		})
	}

	return ret, nil
}

func (s *Service) snapshotTags(ctx context.Context, ids []int) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	tags, err := s.Tag.FindMany(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("finding tags: %w", err)
	}

	var ret []string
	for _, t := range tags {
		if t != nil {
			ret = append(ret, t.Name)
		}
	}

	return ret, nil
}

func toTombstoneStashIDs(ids []models.StashID) []models.TombstoneStashID {
	if len(ids) == 0 {
		return nil
	}

	ret := make([]models.TombstoneStashID, len(ids))
	for i, id := range ids {
		ret[i] = models.TombstoneStashID{Endpoint: id.Endpoint, StashID: id.StashID}
	}
	return ret
}

func toTombstoneFile(f *models.VideoFile) *models.TombstoneFile {
	fps := f.Fingerprints
	ret := &models.TombstoneFile{
		Basename:   f.Basename,
		Path:       f.Path,
		Size:       f.Size,
		Duration:   f.Duration,
		Width:      f.Width,
		Height:     f.Height,
		VideoCodec: f.VideoCodec,
		BitRate:    f.BitRate,
		Oshash:     fps.GetString(models.FingerprintTypeOshash),
		MD5:        fps.GetString(models.FingerprintTypeMD5),
	}

	// Fingerprints.GetInt64 is inverted upstream, so read the phash directly.
	if fp := fps.For(models.FingerprintTypePhash); fp != nil {
		if v, ok := fp.Fingerprint.(int64); ok {
			ret.Phash = &v
		}
	}

	return ret
}

func customFieldString(fields map[string]interface{}, key string) string {
	s, _ := fields[key].(string)
	return s
}
