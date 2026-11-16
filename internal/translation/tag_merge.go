package translation

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/stashapp/stash/pkg/models"
)

// TagMergeResult describes what MergeTag did besides the upstream merge.
type TagMergeResult struct {
	SourceID        int
	SourceName      string
	DestinationID   int
	DestinationName string

	// MovedStashIDs were added to the destination.
	MovedStashIDs []models.StashID
	// DuplicateStashIDs were already on the destination and dropped.
	DuplicateStashIDs []models.StashID

	// AddedParentIDs and AddedChildIDs are the source's relations that were
	// added to the destination.
	AddedParentIDs []int
	AddedChildIDs  []int
	// SkippedRelations describes source relations that were not added
	// because they would create a self-reference or a cycle.
	SkippedRelations []string

	Warnings []string
}

func (r *TagMergeResult) String() string {
	parts := []string{fmt.Sprintf("merged tag %q (%d) into %q (%d)", r.SourceName, r.SourceID, r.DestinationName, r.DestinationID)}
	if len(r.MovedStashIDs) > 0 {
		parts = append(parts, "moved stash IDs "+formatStashIDs(r.MovedStashIDs))
	}
	if len(r.DuplicateStashIDs) > 0 {
		parts = append(parts, "dropped duplicate stash IDs "+formatStashIDs(r.DuplicateStashIDs))
	}
	if len(r.AddedParentIDs) > 0 {
		parts = append(parts, fmt.Sprintf("added parents %v", r.AddedParentIDs))
	}
	if len(r.AddedChildIDs) > 0 {
		parts = append(parts, fmt.Sprintf("added children %v", r.AddedChildIDs))
	}
	if len(r.SkippedRelations) > 0 {
		parts = append(parts, "skipped relations: "+strings.Join(r.SkippedRelations, "; "))
	}
	return strings.Join(parts, ", ")
}

func formatStashIDs(ids []models.StashID) string {
	s := make([]string, len(ids))
	for i, id := range ids {
		s[i] = id.Endpoint + " " + id.StashID
	}
	return "[" + strings.Join(s, ", ") + "]"
}

// MergeTag merges the source tag into the destination with the upstream
// TagStore.Merge and closes the gaps it leaves for translation merges:
//
//   - Merge drops the source's parents and children. They are added to the
//     destination, except relations that would create a self-reference or a
//     cycle.
//   - Merge moves stash IDs without de-duplication. The destination keeps its
//     own stash IDs and gets the source's that it does not have yet. A
//     different stash ID for an endpoint the destination already has is kept
//     too, with a warning.
//
// It must be called inside a write transaction, so a failure rolls back the
// whole merge.
func MergeTag(ctx context.Context, qb models.TagReaderWriter, sourceID, destinationID int) (*TagMergeResult, error) {
	if sourceID == destinationID {
		return nil, fmt.Errorf("cannot merge tag %d into itself", sourceID)
	}

	src, err := findTag(ctx, qb, sourceID)
	if err != nil {
		return nil, err
	}
	dst, err := findTag(ctx, qb, destinationID)
	if err != nil {
		return nil, err
	}

	res := &TagMergeResult{SourceID: src.ID, SourceName: src.Name, DestinationID: dst.ID, DestinationName: dst.Name}

	// read everything Merge loses before calling it
	srcParents, err := qb.GetParentIDs(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	srcChildren, err := qb.GetChildIDs(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	srcStashIDs, err := qb.GetStashIDs(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	dstStashIDs, err := qb.GetStashIDs(ctx, destinationID)
	if err != nil {
		return nil, err
	}

	if err := qb.Merge(ctx, []int{sourceID}, destinationID); err != nil {
		return nil, err
	}

	if err := mergeStashIDs(ctx, qb, res, dstStashIDs, srcStashIDs); err != nil {
		return nil, err
	}
	if err := mergeRelations(ctx, qb, res, srcParents, srcChildren); err != nil {
		return nil, err
	}

	return res, nil
}

func findTag(ctx context.Context, qb models.TagGetter, id int) (*models.Tag, error) {
	t, err := qb.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, fmt.Errorf("tag %d not found", id)
	}
	return t, nil
}

// mergeStashIDs replaces the destination's stash IDs, which Merge may have
// filled with duplicates, with its own plus the source's new ones.
func mergeStashIDs(ctx context.Context, qb models.TagReaderWriter, res *TagMergeResult, dst, src []models.StashID) error {
	if len(src) == 0 {
		return nil
	}

	merged := slices.Clone(dst)
	for _, s := range src {
		identical := slices.ContainsFunc(merged, func(m models.StashID) bool {
			return m.Endpoint == s.Endpoint && m.StashID == s.StashID
		})
		if identical {
			res.DuplicateStashIDs = append(res.DuplicateStashIDs, s)
			continue
		}

		sameEndpoint := slices.ContainsFunc(merged, func(m models.StashID) bool { return m.Endpoint == s.Endpoint })
		if sameEndpoint {
			res.Warnings = append(res.Warnings, fmt.Sprintf("tag %q (%d) now has several stash IDs for %s; the stash ID %s came from %q (%d)",
				res.DestinationName, res.DestinationID, s.Endpoint, s.StashID, res.SourceName, res.SourceID))
		}
		merged = append(merged, s)
		res.MovedStashIDs = append(res.MovedStashIDs, s)
	}

	partial := models.NewTagPartial()
	partial.StashIDs = &models.UpdateStashIDs{StashIDs: merged, Mode: models.RelationshipUpdateModeSet}
	_, err := qb.UpdatePartial(ctx, res.DestinationID, partial)
	return err
}

// mergeRelations adds the source's parents and children to the destination
// one at a time, checking each against the hierarchy as it is at that point.
func mergeRelations(ctx context.Context, qb models.TagReaderWriter, res *TagMergeResult, parents, children []int) error {
	dstID := res.DestinationID

	if len(parents) > 0 || len(children) > 0 {
		// the merge deleted the source, so these are the destination's own
		existing, err := qb.GetParentIDs(ctx, dstID)
		if err != nil {
			return err
		}
		parents = slices.DeleteFunc(slices.Clone(parents), func(id int) bool { return slices.Contains(existing, id) })

		existing, err = qb.GetChildIDs(ctx, dstID)
		if err != nil {
			return err
		}
		children = slices.DeleteFunc(slices.Clone(children), func(id int) bool { return slices.Contains(existing, id) })
	}

	skip := func(relation string, id int, reason string) error {
		name := fmt.Sprint(id)
		if t, err := qb.Find(ctx, id); err != nil {
			return err
		} else if t != nil {
			name = fmt.Sprintf("%q (%d)", t.Name, id)
		}
		res.SkippedRelations = append(res.SkippedRelations, fmt.Sprintf("%s %s: %s", relation, name, reason))
		return nil
	}

	// add adds each id as a relation of the destination unless it is the
	// destination itself or found by lookup (the destination's descendants
	// for a parent, its ancestors for a child), which would close a cycle.
	add := func(label string, ids []int, lookup func(context.Context, int, []int) ([]*models.TagPath, error), set func(*models.TagPartial, *models.UpdateIDs), added *[]int) error {
		for _, id := range ids {
			if id == dstID {
				if err := skip(label, id, "would be a self-reference"); err != nil {
					return err
				}
				continue
			}
			related, err := lookup(ctx, dstID, nil)
			if err != nil {
				return err
			}
			if slices.ContainsFunc(related, func(t *models.TagPath) bool { return t.ID == id }) {
				if err := skip(label, id, "would create a cycle"); err != nil {
					return err
				}
				continue
			}

			partial := models.NewTagPartial()
			set(&partial, &models.UpdateIDs{IDs: []int{id}, Mode: models.RelationshipUpdateModeAdd})
			if _, err := qb.UpdatePartial(ctx, dstID, partial); err != nil {
				return err
			}
			*added = append(*added, id)
		}
		return nil
	}

	// a descendant of the destination as its parent closes a loop
	if err := add("parent", parents, qb.FindAllDescendants, func(p *models.TagPartial, ids *models.UpdateIDs) { p.ParentIDs = ids }, &res.AddedParentIDs); err != nil {
		return err
	}
	// an ancestor of the destination as its child closes a loop
	return add("child", children, qb.FindAllAncestors, func(p *models.TagPartial, ids *models.UpdateIDs) { p.ChildIDs = ids }, &res.AddedChildIDs)
}
