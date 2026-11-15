package api

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/translate"
)

// translationTestSample is translated by testTranslation. Japanese, because
// that is the main use case.
const (
	translationTestSample     = "夏の日の思い出"
	translationTestSampleLang = "ja"
	translationTestTimeout    = 90 * time.Second
)

var languageCodeRE = regexp.MustCompile(`^[A-Za-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)

func translationProviderToGQL(p translate.ProviderType) TranslationProvider {
	switch p {
	case translate.ProviderOpenAICompatible:
		return TranslationProviderOpenaiCompatible
	default:
		return TranslationProviderNone
	}
}

func translationProviderFromGQL(p TranslationProvider) translate.ProviderType {
	switch p {
	case TranslationProviderOpenaiCompatible:
		return translate.ProviderOpenAICompatible
	default:
		return translate.ProviderNone
	}
}

func makeConfigTranslationResult() *ConfigTranslationResult {
	c := config.GetInstance()

	mappings := c.GetTranslationLanguageMappings()
	retMappings := make([]*TranslationLanguageMapping, len(mappings))
	for i, m := range mappings {
		retMappings[i] = &TranslationLanguageMapping{TagID: m.TagID, LanguageCode: m.LanguageCode}
	}

	openai := c.GetTranslationOpenAIConfig()

	return &ConfigTranslationResult{
		Provider:           translationProviderToGQL(c.GetTranslationProvider()),
		TargetLanguage:     c.GetTranslationTargetLanguage(),
		ApplyWithoutReview: c.GetTranslationApplyWithoutReview(),
		LanguageMappings:   retMappings,
		Openai: &ConfigTranslationOpenAIResult{
			BaseURL:      openai.BaseURL,
			Model:        openai.Model,
			APIKey:       openai.APIKey,
			SystemPrompt: openai.SystemPrompt,
		},
		DefaultSystemPrompt: translate.DefaultSystemPrompt,
	}
}

func normalizeLanguageCode(code string) (string, error) {
	code = strings.TrimSpace(code)
	if !languageCodeRE.MatchString(code) {
		return "", fmt.Errorf("invalid language code: %q", code)
	}
	return code, nil
}

func validateLanguageMappings(input []*TranslationLanguageMappingInput) ([]config.TranslationLanguageMapping, error) {
	ret := make([]config.TranslationLanguageMapping, 0, len(input))
	seen := make(map[string]bool)

	for _, m := range input {
		tagID := strings.TrimSpace(m.TagID)
		if tagID == "" {
			return nil, fmt.Errorf("language mapping without tag")
		}
		if seen[tagID] {
			return nil, fmt.Errorf("tag %s is mapped more than once", tagID)
		}
		seen[tagID] = true

		code, err := normalizeLanguageCode(m.LanguageCode)
		if err != nil {
			return nil, err
		}

		ret = append(ret, config.TranslationLanguageMapping{TagID: tagID, LanguageCode: code})
	}

	return ret, nil
}

func (r *mutationResolver) ConfigureTranslation(ctx context.Context, input ConfigTranslationInput) (*ConfigTranslationResult, error) {
	c := config.GetInstance()

	// validate everything before setting anything, so an invalid input does
	// not leave a partly applied configuration in memory
	var targetLanguage string
	if input.TargetLanguage != nil {
		code, err := normalizeLanguageCode(*input.TargetLanguage)
		if err != nil {
			return makeConfigTranslationResult(), err
		}
		targetLanguage = code
	}

	var mappings []config.TranslationLanguageMapping
	if input.LanguageMappings != nil {
		var err error
		mappings, err = validateLanguageMappings(input.LanguageMappings)
		if err != nil {
			return makeConfigTranslationResult(), err
		}
	}

	if input.Provider != nil {
		c.SetString(config.TranslationProvider, string(translationProviderFromGQL(*input.Provider)))
	}

	if input.TargetLanguage != nil {
		c.SetString(config.TranslationTargetLanguage, targetLanguage)
	}

	r.setConfigBool(config.TranslationApplyWithoutReview, input.ApplyWithoutReview)

	if input.LanguageMappings != nil {
		c.SetTranslationLanguageMappings(mappings)
	}

	if o := input.Openai; o != nil {
		r.setConfigString(config.TranslationOpenAIBaseURL, trimmed(o.BaseURL))
		r.setConfigString(config.TranslationOpenAIModel, trimmed(o.Model))
		r.setConfigString(config.TranslationOpenAIAPIKey, trimmed(o.APIKey))
		r.setConfigString(config.TranslationOpenAISystemPrompt, o.SystemPrompt)
	}

	if err := c.Write(); err != nil {
		return makeConfigTranslationResult(), err
	}

	return makeConfigTranslationResult(), nil
}

func trimmed(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	return &v
}

// overlayTranslationInput applies the fields set in input on top of the saved
// translation settings, without saving anything.
func overlayTranslationInput(cfg translate.Config, targetLanguage string, input ConfigTranslationInput) (translate.Config, string) {
	if input.Provider != nil {
		cfg.Provider = translationProviderFromGQL(*input.Provider)
	}
	if input.TargetLanguage != nil {
		targetLanguage = strings.TrimSpace(*input.TargetLanguage)
	}
	if o := input.Openai; o != nil {
		if o.BaseURL != nil {
			cfg.OpenAI.BaseURL = *o.BaseURL
		}
		if o.Model != nil {
			cfg.OpenAI.Model = *o.Model
		}
		if o.APIKey != nil {
			cfg.OpenAI.APIKey = strings.TrimSpace(*o.APIKey)
		}
		if o.SystemPrompt != nil {
			cfg.OpenAI.SystemPrompt = *o.SystemPrompt
		}
	}
	return cfg, targetLanguage
}

func (r *queryResolver) TestTranslation(ctx context.Context, input ConfigTranslationInput) (*TranslationTestResult, error) {
	c := config.GetInstance()
	cfg, target := overlayTranslationInput(c.GetTranslationConfig(), c.GetTranslationTargetLanguage(), input)

	ret := &TranslationTestResult{Source: translationTestSample}
	fail := func(err error) (*TranslationTestResult, error) {
		msg := err.Error()
		ret.Error = &msg
		return ret, nil
	}

	tr, err := translate.New(cfg, nil)
	if err != nil {
		return fail(err)
	}

	ctx, cancel := context.WithTimeout(ctx, translationTestTimeout)
	defer cancel()

	out, err := tr.Translate(ctx, []translate.Text{{Field: translate.FieldTitle, Value: translationTestSample}}, translationTestSampleLang, target)
	if err != nil {
		return fail(err)
	}

	ret.Success = true
	ret.Translation = &out[0]
	return ret, nil
}
