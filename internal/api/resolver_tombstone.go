package api

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/scene"
	"github.com/stashapp/stash/pkg/utils"
)

func (r *Resolver) Tombstone() TombstoneResolver {
	return &tombstoneResolver{r}
}

func (r *Resolver) TombstoneFile() TombstoneFileResolver {
	return &tombstoneFileResolver{r}
}

func (r *Resolver) TombstonePerformer() TombstonePerformerResolver {
	return &tombstonePerformerResolver{r}
}

type tombstoneResolver struct{ *Resolver }
type tombstoneFileResolver struct{ *Resolver }
type tombstonePerformerResolver struct{ *Resolver }

func (r *tombstoneResolver) Date(ctx context.Context, obj *models.Tombstone) (*string, error) {
	if obj.Date == nil {
		return nil, nil
	}
	ret := obj.Date.String()
	return &ret, nil
}

func (r *tombstoneResolver) Rating100(ctx context.Context, obj *models.Tombstone) (*int, error) {
	return obj.Rating, nil
}

func (r *tombstoneResolver) TagItems(ctx context.Context, obj *models.Tombstone) ([]*TombstoneTag, error) {
	ret := make([]*TombstoneTag, len(obj.Tags))
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		for i, name := range obj.Tags {
			tag, err := scene.FindTombstoneTag(ctx, r.repository.Tag, name)
			if err != nil {
				return err
			}
			ret[i] = &TombstoneTag{Name: name, Tag: tag}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return ret, nil
}

func (r *tombstonePerformerResolver) Performer(ctx context.Context, obj *models.TombstonePerformer) (ret *models.Performer, err error) {
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ret, err = scene.FindTombstonePerformer(ctx, r.repository.Performer, *obj)
		return err
	}); err != nil {
		return nil, err
	}
	return ret, nil
}

func (r *tombstoneFileResolver) Phash(ctx context.Context, obj *models.TombstoneFile) (*string, error) {
	if obj.Phash == nil {
		return nil, nil
	}
	ret := utils.PhashToString(*obj.Phash)
	return &ret, nil
}

// tombstoneBasenameTerm normalises a Filename search term like a stored
// Filename stem: it strips a trailing extension if it is one of videoExts
// (ignoring case), then a trailing resolution token. Other dotted parts are
// kept, since release names such as "Studio.24.01.01" are full of dots. If
// nothing is left, as for ".mkv", the term is searched as given.
func tombstoneBasenameTerm(term string, videoExts []string) string {
	stem := term
	if ext := strings.TrimPrefix(filepath.Ext(term), "."); ext != "" {
		for _, v := range videoExts {
			if strings.EqualFold(ext, strings.TrimPrefix(v, ".")) {
				stem = strings.TrimSuffix(term, "."+ext)
				break
			}
		}
	}

	if ret := models.NormaliseTombstoneStem(stem); ret != "" {
		return ret
	}
	return term
}

func (r *queryResolver) FindTombstones(ctx context.Context, tombstoneFilter *TombstoneFilterInput, filter *models.FindFilterType) (ret *models.TombstoneQueryResult, err error) {
	var f *models.TombstoneFilter
	if tombstoneFilter != nil {
		f = &models.TombstoneFilter{Origin: tombstoneFilter.Origin}
		if tombstoneFilter.URL != nil {
			f.URL = *tombstoneFilter.URL
		}
		if tombstoneFilter.StashID != nil {
			f.StashID = *tombstoneFilter.StashID
		}
		if tombstoneFilter.Oshash != nil {
			f.Oshash = *tombstoneFilter.Oshash
		}
		if tombstoneFilter.Md5 != nil {
			f.MD5 = *tombstoneFilter.Md5
		}
		if tombstoneFilter.Code != nil {
			f.Code = *tombstoneFilter.Code
		}
		if tombstoneFilter.Title != nil {
			f.Title = *tombstoneFilter.Title
		}
		if tombstoneFilter.Basename != nil {
			f.Basename = tombstoneBasenameTerm(*tombstoneFilter.Basename, config.GetInstance().GetVideoExtensions())
		}
		if p := tombstoneFilter.Phash; p != nil {
			value, err := utils.StringToPhash(p.Value)
			if err != nil {
				return nil, fmt.Errorf("invalid phash %q: %w", p.Value, err)
			}
			f.Phash = &models.TombstonePhashInput{Value: value}
			if p.Distance != nil {
				f.Phash.Distance = *p.Distance
			}
		}
	}

	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.Tombstone.Query(ctx, f, filter)
		return err
	}); err != nil {
		return nil, err
	}
	return ret, nil
}

func (r *mutationResolver) TombstoneDestroy(ctx context.Context, id string) (bool, error) {
	tombstoneID, err := strconv.Atoi(id)
	if err != nil {
		return false, fmt.Errorf("converting id: %w", err)
	}

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		return r.repository.Tombstone.Destroy(ctx, tombstoneID)
	}); err != nil {
		return false, err
	}
	return true, nil
}

func (r *mutationResolver) TombstoneUpdate(ctx context.Context, input TombstoneUpdateInput) (ret *models.Tombstone, err error) {
	tombstoneID, err := strconv.Atoi(input.ID)
	if err != nil {
		return nil, fmt.Errorf("converting id: %w", err)
	}

	reason := ""
	if input.Reason != nil {
		reason = strings.TrimSpace(*input.Reason)
	}

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		if err := r.repository.Tombstone.UpdateReason(ctx, tombstoneID, reason); err != nil {
			return err
		}
		ret, err = r.repository.Tombstone.Find(ctx, tombstoneID)
		return err
	}); err != nil {
		return nil, err
	}
	return ret, nil
}
