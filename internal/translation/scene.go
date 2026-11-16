// Package translation machine-translates scene metadata and tag names with
// the active translation provider. Scenes keep the original metadata in
// custom fields (see docs/adr/0001-original-metadata-in-custom-fields.md);
// tags keep their old name as an alias.
package translation

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/translate"
)

// Custom field keys holding the original metadata of a translated scene.
const (
	CustomFieldOriginalTitle   = "original_title"
	CustomFieldOriginalDetails = "original_details"
	CustomFieldSourceLanguage  = "translation_source_lang"
	CustomFieldProvider        = "translation_provider"
	CustomFieldTranslatedAt    = "translated_at"
)

var allCustomFields = []string{
	CustomFieldOriginalTitle,
	CustomFieldOriginalDetails,
	CustomFieldSourceLanguage,
	CustomFieldProvider,
	CustomFieldTranslatedAt,
}

// Options controls which scenes, fields and tags are translated.
type Options struct {
	// LanguageMappings maps language tag IDs to language codes.
	LanguageMappings map[int]string
	TargetLanguage   string
	// Force re-translates fields from their original metadata, even if they
	// are no longer in the source language.
	Force bool
	// MergeOnCollision merges a tag into the tag whose name or alias matches
	// its translation. Otherwise the tag is skipped.
	MergeOnCollision bool
}

// FieldTranslation is the translation of one scene field.
type FieldTranslation struct {
	// Source is the text that was translated. It becomes the original
	// metadata when the translation is applied.
	Source string
	// Current is the field's value before the translation.
	Current     string
	Translation string
	Language    string
}

// SceneTranslation is the translation of a scene's title and details. A nil
// field was not translated.
type SceneTranslation struct {
	SceneID  int
	Provider string
	Title    *FieldTranslation
	Details  *FieldTranslation
	Warnings []string
}

// Empty reports whether nothing was translated.
func (t *SceneTranslation) Empty() bool {
	return t.Title == nil && t.Details == nil
}

// SourceLanguage returns the language code of the translated fields, joined
// with "," in the unlikely case that title and details differ.
func (t *SceneTranslation) SourceLanguage() string {
	var langs []string
	for _, f := range []*FieldTranslation{t.Title, t.Details} {
		if f != nil && (len(langs) == 0 || langs[0] != f.Language) {
			langs = append(langs, f.Language)
		}
	}
	return strings.Join(langs, ",")
}

type performerName struct {
	name    string
	aliases []string
}

// sceneData is what planning needs from the database.
type sceneData struct {
	title        string
	details      string
	tagIDs       []int
	customFields map[string]interface{}
	performers   []performerName
}

// newOriginalMinShare is the share of source-script letters a field needs to
// replace an already stored original.
const newOriginalMinShare = 0.5

type fieldPlan struct {
	source   string
	current  string
	language string
}

type scenePlan struct {
	title    *fieldPlan
	details  *fieldPlan
	warnings []string
}

func customFieldString(fields map[string]interface{}, key string) string {
	if v, ok := fields[key].(string); ok {
		return v
	}
	return ""
}

// sceneLanguages returns the language codes of the scene's mapped language
// tags, sorted and without duplicates.
func sceneLanguages(tagIDs []int, mappings map[int]string) []string {
	seen := make(map[string]bool)
	var ret []string
	for _, id := range tagIDs {
		if code, ok := mappings[id]; ok && !seen[code] {
			seen[code] = true
			ret = append(ret, code)
		}
	}
	sort.Strings(ret)
	return ret
}

// planField decides whether and from which text a field is translated:
//
//   - If the current value is in a source language, it is the new original.
//     This also covers scenes where Identify wrote source-language text again.
//     If an original is already stored, the current value must be mostly in
//     the source script: a translation with a leftover source-language word
//     must not replace the original.
//   - Otherwise, with force, the stored original is translated again.
//   - Otherwise the field is skipped.
func planField(name, current, original, storedLang string, langs []string, force bool) (*fieldPlan, string) {
	lang, err := translate.DetectLanguage(current, langs)
	switch {
	case errors.Is(err, translate.ErrAmbiguousLanguage):
		return nil, fmt.Sprintf("%s: source language is ambiguous between %s", name, strings.Join(langs, ", "))
	case lang != "" && (original == "" || translate.SourceScriptShare(current, lang) >= newOriginalMinShare):
		return &fieldPlan{source: current, current: current, language: lang}, ""
	case lang != "" && !force:
		return nil, fmt.Sprintf("%s: keeps the stored original, the current value is mostly translated but still contains %s text", name, lang)
	case !force || original == "":
		return nil, ""
	}

	// force: translate the stored original again
	lang = storedLang
	if lang == "" || strings.Contains(lang, ",") {
		lang, err = translate.DetectLanguage(original, langs)
		if err != nil {
			return nil, fmt.Sprintf("%s: source language of the original is ambiguous between %s", name, strings.Join(langs, ", "))
		}
		if lang == "" {
			return nil, ""
		}
	}
	return &fieldPlan{source: original, current: current, language: lang}, ""
}

func planScene(d sceneData, opts Options) scenePlan {
	var p scenePlan

	langs := sceneLanguages(d.tagIDs, opts.LanguageMappings)
	if len(langs) == 0 {
		return p
	}
	if len(langs) > 1 {
		p.warnings = append(p.warnings, fmt.Sprintf("scene has several language tags: %s", strings.Join(langs, ", ")))
	}

	storedLang := customFieldString(d.customFields, CustomFieldSourceLanguage)

	var warning string
	p.title, warning = planField("title", d.title, customFieldString(d.customFields, CustomFieldOriginalTitle), storedLang, langs, opts.Force)
	if warning != "" {
		p.warnings = append(p.warnings, warning)
	}
	p.details, warning = planField("details", d.details, customFieldString(d.customFields, CustomFieldOriginalDetails), storedLang, langs, opts.Force)
	if warning != "" {
		p.warnings = append(p.warnings, warning)
	}

	return p
}

// titleContext builds the context sent along with a title: the known
// performer names, so the provider does not have to guess name readings, and
// the details in the source language, to resolve ambiguity.
func titleContext(d sceneData, p scenePlan) string {
	var lines []string

	if len(d.performers) > 0 {
		names := make([]string, len(d.performers))
		for i, perf := range d.performers {
			names[i] = perf.name
			if len(perf.aliases) > 0 {
				names[i] += " (also: " + strings.Join(perf.aliases, ", ") + ")"
			}
		}
		lines = append(lines, "Performers: "+strings.Join(names, "; "))
	}

	details := d.details
	switch {
	case p.details != nil:
		details = p.details.source
	case customFieldString(d.customFields, CustomFieldOriginalDetails) != "":
		details = customFieldString(d.customFields, CustomFieldOriginalDetails)
	}
	if strings.TrimSpace(details) != "" {
		lines = append(lines, "Details: "+details)
	}

	return strings.Join(lines, "\n")
}

// maxAliases limits the aliases per performer sent as context.
const maxAliases = 5

func loadSceneData(ctx context.Context, r models.Repository, sceneID int) (*sceneData, error) {
	s, err := r.Scene.Find(ctx, sceneID)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, fmt.Errorf("scene %d not found", sceneID)
	}

	if err := s.LoadTagIDs(ctx, r.Scene); err != nil {
		return nil, err
	}
	if err := s.LoadPerformerIDs(ctx, r.Scene); err != nil {
		return nil, err
	}

	cf, err := r.Scene.GetCustomFields(ctx, sceneID)
	if err != nil {
		return nil, err
	}

	d := &sceneData{
		title:        s.Title,
		details:      s.Details,
		tagIDs:       s.TagIDs.List(),
		customFields: cf,
	}

	performers, err := r.Performer.FindMany(ctx, s.PerformerIDs.List())
	if err != nil {
		return nil, err
	}
	for _, p := range performers {
		if err := p.LoadAliases(ctx, r.Performer); err != nil {
			return nil, err
		}
		aliases := p.Aliases.List()
		if len(aliases) > maxAliases {
			aliases = aliases[:maxAliases]
		}
		d.performers = append(d.performers, performerName{name: p.Name, aliases: aliases})
	}

	return d, nil
}

// Translate translates a scene without saving anything. The database is only
// read inside a short read transaction; the provider is called outside of
// it, because translation can take a long time.
func Translate(ctx context.Context, r models.Repository, tr translate.Translator, sceneID int, opts Options) (*SceneTranslation, error) {
	var d *sceneData
	if err := r.WithReadTxn(ctx, func(ctx context.Context) error {
		var err error
		d, err = loadSceneData(ctx, r, sceneID)
		return err
	}); err != nil {
		return nil, err
	}

	p := planScene(*d, opts)
	ret := &SceneTranslation{SceneID: sceneID, Provider: tr.Name(), Warnings: p.warnings}
	logger.TraceFunc(func() (string, []interface{}) {
		return "Translation plan for scene %d (languages %v, force %t): title: %s; details: %s", []interface{}{
			sceneID, sceneLanguages(d.tagIDs, opts.LanguageMappings), opts.Force, describePlan(p.title), describePlan(p.details),
		}
	})

	translateField := func(fp *fieldPlan, text translate.Text) (*FieldTranslation, error) {
		if fp == nil {
			return nil, nil
		}
		start := time.Now()
		out, err := tr.Translate(ctx, []translate.Text{text}, fp.language, opts.TargetLanguage)
		if err != nil {
			return nil, err
		}
		logger.Debugf("Translated %s of scene %d from %s with %s in %s", text.Field, sceneID, fp.language, tr.Name(), time.Since(start).Round(time.Millisecond))
		if l, _ := translate.DetectLanguage(out[0], []string{fp.language}); l != "" {
			ret.Warnings = append(ret.Warnings, fmt.Sprintf("%s: translation still contains %s text", text.Field, fp.language))
		}
		return &FieldTranslation{Source: fp.source, Current: fp.current, Translation: out[0], Language: fp.language}, nil
	}

	var err error
	if p.title != nil {
		ret.Title, err = translateField(p.title, translate.Text{Field: translate.FieldTitle, Value: p.title.source, Context: titleContext(*d, p)})
		if err != nil {
			return nil, fmt.Errorf("translating title: %w", err)
		}
	}
	if p.details != nil {
		ret.Details, err = translateField(p.details, translate.Text{Field: translate.FieldDetails, Value: p.details.source})
		if err != nil {
			return nil, fmt.Errorf("translating details: %w", err)
		}
	}

	for _, w := range ret.Warnings {
		logger.Warnf("Translating scene %d: %s", sceneID, w)
	}
	if ret.Empty() {
		logger.Debugf("Translating scene %d: nothing to translate", sceneID)
	}

	return ret, nil
}

func describePlan(fp *fieldPlan) string {
	switch {
	case fp == nil:
		return "unchanged"
	case fp.source == fp.current:
		return fmt.Sprintf("translate %q from %s", fp.source, fp.language)
	default:
		return fmt.Sprintf("translate the original %q from %s (current %q)", fp.source, fp.language, fp.current)
	}
}

// LogApplied logs a saved translation. Call it after the transaction that
// applied the translation was committed.
func LogApplied(t *SceneTranslation) {
	var fields []string
	if t.Title != nil {
		fields = append(fields, "title")
	}
	if t.Details != nil {
		fields = append(fields, "details")
	}
	logger.Infof("Saved translation of scene %d (%s) from %s with %s", t.SceneID, strings.Join(fields, ", "), t.SourceLanguage(), t.Provider)
}

// LogRestored logs restored original metadata. Call it after the transaction
// was committed.
func LogRestored(sceneID int) {
	logger.Infof("Restored original metadata of scene %d", sceneID)
}

// Apply saves a translation: the translations go into title and details, the
// source texts and translation metadata into custom fields. It must be called
// inside a write transaction. It returns the input for Scene.Update.Post
// hooks and the names of the changed fields.
func Apply(ctx context.Context, r models.Repository, t *SceneTranslation, now time.Time) (*models.SceneUpdateInput, []string, error) {
	if t.Empty() {
		return nil, nil, errors.New("nothing to apply")
	}

	partial := models.NewScenePartial()
	cf := map[string]interface{}{
		CustomFieldSourceLanguage: t.SourceLanguage(),
		CustomFieldProvider:       t.Provider,
		CustomFieldTranslatedAt:   now.UTC().Format(time.RFC3339),
	}
	hookInput := &models.SceneUpdateInput{ID: fmt.Sprint(t.SceneID), CustomFields: &models.CustomFieldsInput{Partial: cf}}
	fields := []string{"custom_fields"}

	if t.Title != nil {
		if strings.TrimSpace(t.Title.Translation) == "" {
			return nil, nil, errors.New("title translation is empty")
		}
		partial.Title = models.NewOptionalString(t.Title.Translation)
		cf[CustomFieldOriginalTitle] = t.Title.Source
		hookInput.Title = &t.Title.Translation
		fields = append(fields, "title")
	}
	if t.Details != nil {
		if strings.TrimSpace(t.Details.Translation) == "" {
			return nil, nil, errors.New("details translation is empty")
		}
		partial.Details = models.NewOptionalString(t.Details.Translation)
		cf[CustomFieldOriginalDetails] = t.Details.Source
		hookInput.Details = &t.Details.Translation
		fields = append(fields, "details")
	}

	logger.TraceFunc(func() (string, []interface{}) {
		var changes []string
		for _, f := range []struct {
			name string
			t    *FieldTranslation
		}{{"title", t.Title}, {"details", t.Details}} {
			if f.t != nil {
				changes = append(changes, fmt.Sprintf("%s %q -> %q", f.name, f.t.Current, f.t.Translation))
			}
		}
		return "Writing translation of scene %d: %s; custom fields %v", []interface{}{t.SceneID, strings.Join(changes, "; "), cf}
	})

	if _, err := r.Scene.UpdatePartial(ctx, t.SceneID, partial); err != nil {
		return nil, nil, err
	}
	if err := r.Scene.SetCustomFields(ctx, t.SceneID, models.CustomFieldsInput{Partial: cf}); err != nil {
		return nil, nil, err
	}

	return hookInput, fields, nil
}

// Restore writes the original metadata back into title and details and
// removes the translation custom fields. It must be called inside a write
// transaction. It returns nil if the scene has no original metadata.
func Restore(ctx context.Context, r models.Repository, sceneID int) (*models.SceneUpdateInput, []string, error) {
	cf, err := r.Scene.GetCustomFields(ctx, sceneID)
	if err != nil {
		return nil, nil, err
	}

	hookInput := &models.SceneUpdateInput{ID: fmt.Sprint(sceneID)}
	partial := models.NewScenePartial()
	var fields []string

	if v, ok := cf[CustomFieldOriginalTitle].(string); ok {
		partial.Title = models.NewOptionalString(v)
		hookInput.Title = &v
		fields = append(fields, "title")
	}
	if v, ok := cf[CustomFieldOriginalDetails].(string); ok {
		partial.Details = models.NewOptionalString(v)
		hookInput.Details = &v
		fields = append(fields, "details")
	}

	var remove []string
	for _, k := range allCustomFields {
		if _, ok := cf[k]; ok {
			remove = append(remove, k)
		}
	}

	if len(fields) == 0 && len(remove) == 0 {
		logger.Tracef("Restoring scene %d: no original metadata", sceneID)
		return nil, nil, nil
	}

	logger.TraceFunc(func() (string, []interface{}) {
		var changes []string
		if partial.Title.Set {
			changes = append(changes, fmt.Sprintf("title -> %q", partial.Title.Value))
		}
		if partial.Details.Set {
			changes = append(changes, fmt.Sprintf("details -> %q", partial.Details.Value))
		}
		return "Restoring scene %d: %s; removing custom fields %v", []interface{}{sceneID, strings.Join(changes, "; "), remove}
	})

	if len(fields) > 0 {
		if _, err := r.Scene.UpdatePartial(ctx, sceneID, partial); err != nil {
			return nil, nil, err
		}
	}
	if len(remove) > 0 {
		if err := r.Scene.SetCustomFields(ctx, sceneID, models.CustomFieldsInput{Remove: remove}); err != nil {
			return nil, nil, err
		}
		hookInput.CustomFields = &models.CustomFieldsInput{Remove: remove}
		fields = append(fields, "custom_fields")
	}

	return hookInput, fields, nil
}
