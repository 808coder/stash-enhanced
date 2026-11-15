package api

import (
	"testing"

	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/translate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeLanguageCode(t *testing.T) {
	for _, valid := range []string{"ja", "en", " ru ", "zh-Hans", "pt-BR", "fil"} {
		_, err := normalizeLanguageCode(valid)
		assert.NoError(t, err, valid)
	}

	for _, invalid := range []string{"", "j", "japanese", "ja_JP", "e n", "12"} {
		_, err := normalizeLanguageCode(invalid)
		assert.Error(t, err, invalid)
	}
}

func TestValidateLanguageMappings(t *testing.T) {
	got, err := validateLanguageMappings([]*TranslationLanguageMappingInput{
		{TagID: " 1 ", LanguageCode: "ja"},
		{TagID: "2", LanguageCode: " ru"},
	})
	require.NoError(t, err)
	assert.Equal(t, []config.TranslationLanguageMapping{
		{TagID: "1", LanguageCode: "ja"},
		{TagID: "2", LanguageCode: "ru"},
	}, got)

	_, err = validateLanguageMappings([]*TranslationLanguageMappingInput{{TagID: "", LanguageCode: "ja"}})
	assert.Error(t, err, "missing tag")

	_, err = validateLanguageMappings([]*TranslationLanguageMappingInput{
		{TagID: "1", LanguageCode: "ja"},
		{TagID: "1", LanguageCode: "ru"},
	})
	assert.Error(t, err, "duplicate tag")

	_, err = validateLanguageMappings([]*TranslationLanguageMappingInput{{TagID: "1", LanguageCode: "Japanese"}})
	assert.Error(t, err, "invalid code")
}

func TestOverlayTranslationInput(t *testing.T) {
	saved := translate.Config{
		Provider: translate.ProviderNone,
		OpenAI: translate.OpenAIConfig{
			BaseURL: "http://saved/v1",
			Model:   "saved-model",
			APIKey:  "saved-key",
		},
	}

	provider := TranslationProviderOpenaiCompatible
	model := "new-model"
	target := "de"

	got, gotTarget := overlayTranslationInput(saved, "en", ConfigTranslationInput{
		Provider:       &provider,
		TargetLanguage: &target,
		Openai:         &ConfigTranslationOpenAIInput{Model: &model},
	})

	assert.Equal(t, translate.ProviderOpenAICompatible, got.Provider)
	assert.Equal(t, "de", gotTarget)
	assert.Equal(t, "http://saved/v1", got.OpenAI.BaseURL, "unset fields keep the saved value")
	assert.Equal(t, "new-model", got.OpenAI.Model)
	assert.Equal(t, "saved-key", got.OpenAI.APIKey)

	got, gotTarget = overlayTranslationInput(saved, "en", ConfigTranslationInput{})
	assert.Equal(t, saved, got)
	assert.Equal(t, "en", gotTarget)
}

func TestTranslationProviderMapping(t *testing.T) {
	for _, p := range AllTranslationProvider {
		assert.Equal(t, p, translationProviderToGQL(translationProviderFromGQL(p)))
	}
}
