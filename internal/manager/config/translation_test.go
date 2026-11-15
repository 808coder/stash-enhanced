package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stashapp/stash/pkg/translate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_TranslationDefaults(t *testing.T) {
	i := InitializeEmpty()

	assert.Equal(t, translate.ProviderNone, i.GetTranslationProvider())
	assert.Equal(t, "en", i.GetTranslationTargetLanguage())
	assert.False(t, i.GetTranslationApplyWithoutReview())
	assert.Empty(t, i.GetTranslationLanguageMappings())
}

func TestConfig_TranslationSettings(t *testing.T) {
	i := InitializeEmpty()

	i.SetString(TranslationProvider, string(translate.ProviderOpenAICompatible))
	i.SetString(TranslationOpenAIBaseURL, "http://host:11434/v1")
	i.SetString(TranslationOpenAIModel, "qwen3:14b")
	i.SetString(TranslationOpenAIAPIKey, "key")

	assert.Equal(t, translate.Config{
		Provider: translate.ProviderOpenAICompatible,
		OpenAI: translate.OpenAIConfig{
			BaseURL: "http://host:11434/v1",
			Model:   "qwen3:14b",
			APIKey:  "key",
		},
	}, i.GetTranslationConfig())
}

func TestConfig_TranslationLanguageMappingsRoundTrip(t *testing.T) {
	mappings := []TranslationLanguageMapping{
		{TagID: "12", LanguageCode: "ja"},
		{TagID: "34", LanguageCode: "ru"},
	}

	i := InitializeEmpty()
	i.SetTranslationLanguageMappings(mappings)
	assert.Equal(t, mappings, i.GetTranslationLanguageMappings())

	// written to and read back from a config file
	fn := filepath.Join(t.TempDir(), "config.yml")
	i.SetConfigFile(fn)
	require.NoError(t, i.Write())

	data, err := os.ReadFile(fn)
	require.NoError(t, err)
	assert.Contains(t, string(data), "tag_id: \"12\"")
	assert.Contains(t, string(data), "language_code: ja")

	reloaded := InitializeEmpty()
	reloaded.SetConfigFile(fn)
	require.NoError(t, reloaded.load(fn))
	assert.Equal(t, mappings, reloaded.GetTranslationLanguageMappings())
}
