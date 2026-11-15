package config

import (
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/translate"
)

// Translation options. Kept in their own file to limit merge conflicts with
// upstream config.go.
const (
	TranslationProvider              = "translation.provider"
	TranslationTargetLanguage        = "translation.target_language"
	TranslationTargetLanguageDefault = "en"
	TranslationApplyWithoutReview    = "translation.apply_without_review"
	TranslationLanguageMappings      = "translation.language_mappings"

	TranslationOpenAIBaseURL      = "translation.openai.base_url"
	TranslationOpenAIModel        = "translation.openai.model"
	TranslationOpenAIAPIKey       = "translation.openai.api_key"
	TranslationOpenAISystemPrompt = "translation.openai.system_prompt"
)

// TranslationLanguageMapping maps a language tag (e.g. "Japanese") to the
// language code (e.g. "ja") of the scenes carrying it.
type TranslationLanguageMapping struct {
	TagID        string `json:"tag_id" koanf:"tag_id"`
	LanguageCode string `json:"language_code" koanf:"language_code"`
}

// GetTranslationProvider returns the active translation provider. Empty means
// translation is disabled.
func (i *Config) GetTranslationProvider() translate.ProviderType {
	return translate.ProviderType(i.getString(TranslationProvider))
}

// GetTranslationTargetLanguage returns the language code metadata is
// translated into. Defaults to "en".
func (i *Config) GetTranslationTargetLanguage() string {
	ret := i.getString(TranslationTargetLanguage)
	if ret == "" {
		ret = TranslationTargetLanguageDefault
	}
	return ret
}

// GetTranslationApplyWithoutReview returns true if translating a single
// scene saves directly instead of opening the review dialog.
func (i *Config) GetTranslationApplyWithoutReview() bool {
	return i.getBool(TranslationApplyWithoutReview)
}

func (i *Config) GetTranslationLanguageMappings() []TranslationLanguageMapping {
	var ret []TranslationLanguageMapping
	if err := i.unmarshalKey(TranslationLanguageMappings, &ret); err != nil {
		logger.Warnf("error in unmarshalkey: %v", err)
	}
	return ret
}

// SetTranslationLanguageMappings stores the mappings as plain maps, so they
// are written to the config file with snake_case keys.
func (i *Config) SetTranslationLanguageMappings(mappings []TranslationLanguageMapping) {
	v := make([]map[string]string, len(mappings))
	for idx, m := range mappings {
		v[idx] = map[string]string{
			"tag_id":        m.TagID,
			"language_code": m.LanguageCode,
		}
	}
	i.SetInterface(TranslationLanguageMappings, v)
}

func (i *Config) GetTranslationOpenAIConfig() translate.OpenAIConfig {
	return translate.OpenAIConfig{
		BaseURL:      i.getString(TranslationOpenAIBaseURL),
		Model:        i.getString(TranslationOpenAIModel),
		APIKey:       i.getString(TranslationOpenAIAPIKey),
		SystemPrompt: i.getString(TranslationOpenAISystemPrompt),
	}
}

// GetTranslationConfig returns the settings needed to build the active
// Translator with translate.New.
func (i *Config) GetTranslationConfig() translate.Config {
	return translate.Config{
		Provider: i.GetTranslationProvider(),
		OpenAI:   i.GetTranslationOpenAIConfig(),
	}
}
