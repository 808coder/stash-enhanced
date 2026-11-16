package translation

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/plugin/hook"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
	"github.com/stashapp/stash/pkg/tag"
	"github.com/stashapp/stash/pkg/translate"
)

// mappedLanguages returns the language codes of the language mapping, sorted
// and without duplicates. Tags have no language tag, so these are the
// candidate source languages of a tag name.
func (o Options) mappedLanguages() []string {
	seen := make(map[string]bool)
	var ret []string
	for _, code := range o.LanguageMappings {
		if !seen[code] {
			seen[code] = true
			ret = append(ret, code)
		}
	}
	sort.Strings(ret)
	return ret
}

// TagSourceLanguage returns the source language of a tag name, detected from
// its script and limited to the candidate languages. It returns "" if the
// name contains no script of a candidate or is already in the script of the
// target language, and ErrAmbiguousLanguage if the script fits several
// candidates (e.g. kanji only, with ja and zh mapped).
func TagSourceLanguage(name string, candidates []string, target string) (string, error) {
	// a name in the target script needs no translation, even if a candidate
	// shares that script (e.g. ru and uk)
	if l, _ := translate.DetectLanguage(name, []string{target}); l != "" {
		return "", nil
	}
	return translate.DetectLanguage(name, candidates)
}

// TagAction is the outcome of translating a tag.
type TagAction string

const (
	TagRenamed TagAction = "renamed"
	TagMerged  TagAction = "merged"
	TagSkipped TagAction = "skipped"
)

// TagTranslation is the result of translating one tag.
type TagTranslation struct {
	Action   TagAction
	TagID    int
	OldName  string
	NewName  string
	Language string
	Provider string
	// Aliases of a renamed tag after the rename.
	Aliases []string
	// Merge is set if the tag was merged.
	Merge *TagMergeResult
	// Reason explains a skipped tag.
	Reason string
	// Collision is set if the tag was skipped because its translation is
	// used by another tag and merging is off.
	Collision bool
}

type tagCandidate struct {
	name     string
	language string
	context  string
}

// loadTagCandidate reads a tag and decides whether it is translated. It
// returns nil and a reason if not.
func loadTagCandidate(ctx context.Context, qb models.TagReader, tagID int, opts Options) (*tagCandidate, string, error) {
	t, err := qb.Find(ctx, tagID)
	if err != nil {
		return nil, "", err
	}
	if t == nil {
		return nil, "tag no longer exists", nil
	}

	lang, err := TagSourceLanguage(t.Name, opts.mappedLanguages(), opts.TargetLanguage)
	switch {
	case errors.Is(err, translate.ErrAmbiguousLanguage):
		return nil, fmt.Sprintf("source language is ambiguous between %s", strings.Join(opts.mappedLanguages(), ", ")), nil
	case err != nil:
		return nil, "", err
	case lang == "":
		return nil, "not in a mapped source language", nil
	}

	c := &tagCandidate{name: t.Name, language: lang}
	if d := strings.TrimSpace(t.Description); d != "" {
		c.context = "Description: " + d
	}
	return c, "", nil
}

// findCollision returns the other tag whose name or alias matches name,
// case-insensitively, the same rule as tag.EnsureTagNameUnique.
func findCollision(ctx context.Context, qb models.TagNameFinder, tagID int, name string) (*models.Tag, error) {
	other, err := tag.ByName(ctx, qb, name)
	if err != nil {
		return nil, err
	}
	if other != nil && other.ID != tagID {
		return other, nil
	}

	other, err = tag.ByAlias(ctx, qb, name)
	if err != nil {
		return nil, err
	}
	if other != nil && other.ID != tagID {
		return other, nil
	}
	return nil, nil
}

// TranslateTag translates a tag name and saves the result:
//
//   - Without collision the tag is renamed and keeps its old name as an alias.
//   - If the translation is the name or alias of another tag, the tag is
//     merged into it with MergeTag, or skipped if opts.MergeOnCollision is
//     off.
//
// Collisions are checked when saving, so a tag renamed earlier in the same
// batch is found too. The provider is called outside of any transaction.
func TranslateTag(ctx context.Context, r models.Repository, tr translate.Translator, tagID int, opts Options) (*TagTranslation, error) {
	var c *tagCandidate
	var reason string
	if err := r.WithReadTxn(ctx, func(ctx context.Context) error {
		var err error
		c, reason, err = loadTagCandidate(ctx, r.Tag, tagID, opts)
		return err
	}); err != nil {
		return nil, err
	}

	ret := &TagTranslation{Action: TagSkipped, TagID: tagID, Provider: tr.Name(), Reason: reason}
	if c == nil {
		return ret, nil
	}
	ret.OldName = c.name
	ret.Language = c.language

	out, err := tr.Translate(ctx, []translate.Text{{Field: translate.FieldTag, Value: c.name, Context: c.context}}, c.language, opts.TargetLanguage)
	if err != nil {
		return nil, err
	}
	newName := strings.TrimSpace(out[0])
	ret.NewName = newName

	switch {
	case newName == "":
		return nil, errors.New("translation is empty")
	case strings.EqualFold(newName, c.name):
		ret.Reason = "translation equals the current name"
		return ret, nil
	}
	if l, _ := translate.DetectLanguage(newName, []string{c.language}); l != "" {
		ret.Reason = fmt.Sprintf("translation %q still contains %s text", newName, c.language)
		return ret, nil
	}

	if err := r.WithTxn(ctx, func(ctx context.Context) error {
		return applyTagTranslation(ctx, r.Tag, ret, opts)
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

// applyTagTranslation renames or merges the tag in t. It must be called
// inside a write transaction.
func applyTagTranslation(ctx context.Context, qb models.TagReaderWriter, t *TagTranslation, opts Options) error {
	current, err := qb.Find(ctx, t.TagID)
	if err != nil {
		return err
	}
	if current == nil || current.Name != t.OldName {
		t.Reason = "tag was changed during translation"
		return nil
	}

	other, err := findCollision(ctx, qb, t.TagID, t.NewName)
	if err != nil {
		return err
	}

	if other != nil {
		if !opts.MergeOnCollision {
			t.Reason = fmt.Sprintf("translation %q collides with tag %q (%d)", t.NewName, other.Name, other.ID)
			t.Collision = true
			return nil
		}

		t.Merge, err = MergeTag(ctx, qb, t.TagID, other.ID)
		if err != nil {
			return fmt.Errorf("merging into tag %q (%d): %w", other.Name, other.ID, err)
		}
		t.Action = TagMerged
		return nil
	}

	aliases, err := qb.GetAliases(ctx, t.TagID)
	if err != nil {
		return err
	}
	// the old name becomes an alias; an alias equal to the new name goes
	t.Aliases = stringslice.UniqueExcludeFold(append(aliases, t.OldName), t.NewName)

	partial := models.NewTagPartial()
	partial.Name = models.NewOptionalString(t.NewName)
	partial.Aliases = &models.UpdateStrings{Values: t.Aliases, Mode: models.RelationshipUpdateModeSet}
	if err := tag.ValidateUpdate(ctx, t.TagID, partial, qb); err != nil {
		return err
	}
	if _, err := qb.UpdatePartial(ctx, t.TagID, partial); err != nil {
		return err
	}

	t.Action = TagRenamed
	return nil
}

// LogTagTranslation logs the outcome of a tag translation. Call it after the
// transaction was committed.
func LogTagTranslation(t *TagTranslation) {
	switch t.Action {
	case TagRenamed:
		logger.Infof("Renamed tag %d %q -> %q (from %s with %s); the old name is kept as an alias", t.TagID, t.OldName, t.NewName, t.Language, t.Provider)
	case TagMerged:
		logger.Infof("Translated tag %q (%d) to %q (from %s with %s): %s", t.OldName, t.TagID, t.NewName, t.Language, t.Provider, t.Merge)
		for _, w := range t.Merge.Warnings {
			logger.Warnf("Merging tag %q (%d): %s", t.OldName, t.TagID, w)
		}
	default:
		switch {
		case t.OldName == "":
			logger.Infof("Skipped tag %d: %s", t.TagID, t.Reason)
		case t.Collision:
			logger.Warnf("Skipped tag %q (%d): %s", t.OldName, t.TagID, t.Reason)
		default:
			logger.Infof("Skipped tag %q (%d): %s", t.OldName, t.TagID, t.Reason)
		}
	}
}

// TagHookExecutor runs plugin post hooks. *plugin.Cache implements it.
type TagHookExecutor interface {
	ExecutePostHooks(ctx context.Context, id int, hookType hook.TriggerEnum, input interface{}, inputFields []string)
}

// ExecuteTagHooks runs Tag.Update.Post for a renamed tag and Tag.Merge.Post
// for a merged one. The input mirrors the tagUpdate and tagsMerge mutations.
func ExecuteTagHooks(ctx context.Context, hooks TagHookExecutor, t *TagTranslation) {
	if hooks == nil {
		return
	}
	switch t.Action {
	case TagRenamed:
		input := map[string]interface{}{
			"id":      strconv.Itoa(t.TagID),
			"name":    t.NewName,
			"aliases": t.Aliases,
		}
		hooks.ExecutePostHooks(ctx, t.TagID, hook.TagUpdatePost, input, []string{"name", "aliases"})
	case TagMerged:
		input := map[string]interface{}{
			"source":      []string{strconv.Itoa(t.TagID)},
			"destination": strconv.Itoa(t.Merge.DestinationID),
		}
		hooks.ExecutePostHooks(ctx, t.Merge.DestinationID, hook.TagMergePost, input, nil)
	}
}
