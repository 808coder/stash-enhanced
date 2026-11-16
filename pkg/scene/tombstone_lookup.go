package scene

import (
	"context"
	"fmt"

	"github.com/stashapp/stash/pkg/models"
)

// TombstonePerformerFinder finds the library performers a Tombstone
// performer may still correspond to.
type TombstonePerformerFinder interface {
	models.PerformerGetter
	FindByStashID(ctx context.Context, stashID models.StashID) ([]*models.Performer, error)
	FindByNames(ctx context.Context, names []string, nocase bool) ([]*models.Performer, error)
}

// FindTombstonePerformer returns the library performer that a frozen
// Tombstone performer still corresponds to, or nil when there is none.
func FindTombstonePerformer(ctx context.Context, r TombstonePerformerFinder, p models.TombstonePerformer) (*models.Performer, error) {
	for _, id := range p.StashIDs {
		found, err := r.FindByStashID(ctx, models.StashID{Endpoint: id.Endpoint, StashID: id.StashID})
		if err != nil {
			return nil, fmt.Errorf("finding performer by stash id: %w", err)
		}
		if len(found) > 0 {
			return found[0], nil
		}
	}

	// ids are reused after a performer is deleted, so the name must match too
	if p.ID != 0 {
		found, err := r.Find(ctx, p.ID)
		if err != nil {
			return nil, fmt.Errorf("finding performer %d: %w", p.ID, err)
		}
		if found != nil && found.Name == p.Name {
			return found, nil
		}
	}

	if p.Name == "" {
		return nil, nil
	}
	found, err := r.FindByNames(ctx, []string{p.Name}, true)
	if err != nil {
		return nil, fmt.Errorf("finding performer by name: %w", err)
	}
	if len(found) == 1 {
		return found[0], nil
	}

	return nil, nil
}

// TombstoneTagFinder finds library tags by name.
type TombstoneTagFinder interface {
	FindByName(ctx context.Context, name string, nocase bool) (*models.Tag, error)
}

// FindTombstoneTag returns the library tag named like a frozen Tombstone tag,
// ignoring case, or nil when there is none.
func FindTombstoneTag(ctx context.Context, r TombstoneTagFinder, name string) (*models.Tag, error) {
	tag, err := r.FindByName(ctx, name, true)
	if err != nil {
		return nil, fmt.Errorf("finding tag by name: %w", err)
	}
	return tag, nil
}
