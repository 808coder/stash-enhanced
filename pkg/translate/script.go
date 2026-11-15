package translate

import (
	"errors"
	"strings"
	"unicode"
)

// script is a writing system that identifies a source language.
type script int

const (
	scriptKana script = iota
	scriptHan
	scriptHangul
	scriptCyrillic
	scriptGreek
	scriptArabic
	scriptHebrew
	scriptThai
	scriptDevanagari
)

// languageScripts lists the scripts that identify each language. A text is in
// a language if it contains a character of one of its scripts. Languages
// written in Latin script cannot be detected and are missing on purpose.
var languageScripts = map[string][]script{
	"ja": {scriptKana, scriptHan},
	"zh": {scriptHan},
	"ko": {scriptHangul},
	"ru": {scriptCyrillic},
	"uk": {scriptCyrillic},
	"be": {scriptCyrillic},
	"bg": {scriptCyrillic},
	"mk": {scriptCyrillic},
	"sr": {scriptCyrillic},
	"kk": {scriptCyrillic},
	"el": {scriptGreek},
	"ar": {scriptArabic},
	"fa": {scriptArabic},
	"he": {scriptHebrew},
	"th": {scriptThai},
	"hi": {scriptDevanagari},
}

// ErrAmbiguousLanguage is returned by DetectLanguage when the text matches
// more than one candidate language.
var ErrAmbiguousLanguage = errors.New("ambiguous source language")

// baseLanguage returns the primary subtag of a language code in lower case,
// e.g. "zh" for "zh-Hans".
func baseLanguage(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	if i := strings.IndexByte(code, '-'); i >= 0 {
		code = code[:i]
	}
	return code
}

// IsDetectableLanguage reports whether DetectLanguage can recognise text in
// the language, i.e. whether it has a script of its own.
func IsDetectableLanguage(code string) bool {
	_, ok := languageScripts[baseLanguage(code)]
	return ok
}

func runeScript(r rune) (script, bool) {
	switch {
	// the katakana middle dot is punctuation, also used in non-Japanese text
	case r == '・':
		return 0, false
	case unicode.Is(unicode.Hiragana, r), unicode.Is(unicode.Katakana, r):
		return scriptKana, true
	case unicode.Is(unicode.Han, r):
		return scriptHan, true
	case unicode.Is(unicode.Hangul, r):
		return scriptHangul, true
	case unicode.Is(unicode.Cyrillic, r):
		return scriptCyrillic, true
	case unicode.Is(unicode.Greek, r):
		return scriptGreek, true
	case unicode.Is(unicode.Arabic, r):
		return scriptArabic, true
	case unicode.Is(unicode.Hebrew, r):
		return scriptHebrew, true
	case unicode.Is(unicode.Thai, r):
		return scriptThai, true
	case unicode.Is(unicode.Devanagari, r):
		return scriptDevanagari, true
	}
	return 0, false
}

func textScripts(text string) map[script]bool {
	ret := make(map[script]bool)
	for _, r := range text {
		if s, ok := runeScript(r); ok {
			ret[s] = true
		}
	}
	return ret
}

// DetectLanguage returns the candidate language whose script the text
// contains. It returns "" if the text contains no script of any candidate,
// for example because it is already in Latin script.
//
// If several candidates match, Kana decides for Japanese over other Han
// languages; otherwise ErrAmbiguousLanguage is returned. Candidates are
// language codes such as "ja" or "zh-Hans"; the returned code is the
// candidate as given.
func DetectLanguage(text string, candidates []string) (string, error) {
	found := textScripts(text)
	if len(found) == 0 {
		return "", nil
	}

	var matches []string
	for _, c := range candidates {
		for _, s := range languageScripts[baseLanguage(c)] {
			if found[s] {
				matches = append(matches, c)
				break
			}
		}
	}

	switch {
	case len(matches) <= 1:
		if len(matches) == 1 {
			return matches[0], nil
		}
		return "", nil
	case found[scriptKana]:
		for _, m := range matches {
			if baseLanguage(m) == "ja" {
				return m, nil
			}
		}
	}

	// the same language given twice, e.g. "ja" from two language tags
	first := baseLanguage(matches[0])
	for _, m := range matches[1:] {
		if baseLanguage(m) != first {
			return "", ErrAmbiguousLanguage
		}
	}
	return matches[0], nil
}

// SourceScriptShare returns the share of letters in text that are written in
// a script of the language, from 0 to 1. Text without letters returns 0.
func SourceScriptShare(text, lang string) float64 {
	scripts := make(map[script]bool)
	for _, s := range languageScripts[baseLanguage(lang)] {
		scripts[s] = true
	}

	var letters, inScript int
	for _, r := range text {
		if !unicode.IsLetter(r) {
			continue
		}
		letters++
		if s, ok := runeScript(r); ok && scripts[s] {
			inScript++
		}
	}

	if letters == 0 {
		return 0
	}
	return float64(inScript) / float64(letters)
}
