package translate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		candidates []string
		want       string
		wantErr    error
	}{
		{"japanese kana", "ある夏の日", []string{"ja"}, "ja", nil},
		{"japanese kanji only", "巨乳", []string{"ja"}, "ja", nil},
		{"japanese with code", "SSIS-123 ある夏の日", []string{"ja"}, "ja", nil},
		{"russian", "Летний день", []string{"ru"}, "ru", nil},
		{"korean", "여름날", []string{"ko"}, "ko", nil},
		{"latin only", "SSIS-123 A Summer Day", []string{"ja", "ru"}, "", nil},
		{"empty", "", []string{"ja"}, "", nil},
		{"katakana middle dot alone", "3P・4P", []string{"ja"}, "", nil},
		{"script of no candidate", "Летний день", []string{"ja"}, "", nil},
		{"no candidates", "ある夏の日", nil, "", nil},
		{"two tags, script decides", "Летний день", []string{"ja", "ru"}, "ru", nil},
		{"kana decides ja over zh", "ある夏の日", []string{"zh", "ja"}, "ja", nil},
		{"han only with ja and zh", "巨乳", []string{"zh", "ja"}, "", ErrAmbiguousLanguage},
		{"cyrillic with ru and uk", "Летний день", []string{"ru", "uk"}, "", ErrAmbiguousLanguage},
		{"same language twice", "ある夏の日", []string{"ja", "JA"}, "ja", nil},
		{"region subtag", "夏日", []string{"zh-Hans"}, "zh-Hans", nil},
		{"latin language candidate", "Un día de verano", []string{"es"}, "", nil},
	}

	for _, tt := range tests {
		got, err := DetectLanguage(tt.text, tt.candidates)
		assert.Equal(t, tt.want, got, tt.name)
		assert.Equal(t, tt.wantErr, err, tt.name)
	}
}

func TestIsDetectableLanguage(t *testing.T) {
	for _, c := range []string{"ja", "ru", "ko", "zh-Hans", " JA "} {
		assert.True(t, IsDetectableLanguage(c), c)
	}
	for _, c := range []string{"en", "es", "de", "", "xx"} {
		assert.False(t, IsDetectableLanguage(c), c)
	}
}
