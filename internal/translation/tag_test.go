package translation

import (
	"testing"

	"github.com/stashapp/stash/pkg/translate"
	"github.com/stretchr/testify/assert"
)

func TestTagSourceLanguage(t *testing.T) {
	tests := []struct {
		name       string
		tag        string
		candidates []string
		target     string
		want       string
		wantErr    error
	}{
		{"kanji", "中出し", []string{"ja", "ru"}, "en", "ja", nil},
		{"kanji only", "巨乳", []string{"ja"}, "en", "ja", nil},
		{"katakana", "コスプレ", []string{"ja"}, "en", "ja", nil},
		{"cyrillic", "Кремпай", []string{"ja", "ru"}, "en", "ru", nil},
		{"hangul", "코스프레", []string{"ko"}, "en", "ko", nil},
		{"mixed with latin", "4K 中出し", []string{"ja"}, "en", "ja", nil},
		{"latin", "Creampie", []string{"ja", "ru"}, "en", "", nil},
		{"script of an unmapped language", "Кремпай", []string{"ja"}, "en", "", nil},
		{"hangul without ko mapped", "코스프레", []string{"ja"}, "en", "", nil},
		{"no mapping", "中出し", nil, "en", "", nil},
		{"kanji only with ja and zh", "巨乳", []string{"ja", "zh"}, "en", "", translate.ErrAmbiguousLanguage},
		{"kana decides for ja over zh", "中出し", []string{"ja", "zh"}, "en", "ja", nil},
		{"already in the target script", "Кремпай", []string{"uk"}, "ru", "", nil},
		{"han in a han target", "中出し", []string{"ja"}, "zh", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TagSourceLanguage(tt.tag, tt.candidates, tt.target)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMappedLanguages(t *testing.T) {
	opts := Options{LanguageMappings: map[int]string{1: "ru", 2: "ja", 3: "ja"}}
	assert.Equal(t, []string{"ja", "ru"}, opts.mappedLanguages())
	assert.Empty(t, Options{}.mappedLanguages())
}
