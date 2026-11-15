// Package translate provides machine translation of metadata text through
// interchangeable translation providers.
package translate

import (
	"context"
	"errors"
	"net/http"
	"time"
)

// Field tells a translation provider what kind of metadata a text is, so it
// can pick a fitting style (e.g. a short tag name versus a long description).
type Field string

const (
	FieldTitle   Field = "title"
	FieldDetails Field = "details"
	FieldTag     Field = "tag"
)

// Text is a single piece of metadata to translate.
type Text struct {
	Field Field
	Value string
}

// Translator translates metadata text from one language to another.
//
// Translate returns one translation per input text, in input order. from and
// to are language codes such as "ja" or "en". Empty input values translate to
// empty strings. On failure the whole call fails; callers that need per-text
// error handling translate one text per call.
type Translator interface {
	Translate(ctx context.Context, texts []Text, from, to string) ([]string, error)

	// Name identifies the provider and its model, e.g.
	// "openai_compatible:qwen3:14b". It is stored with translated metadata.
	Name() string
}

// ProviderType identifies a translation provider implementation.
type ProviderType string

const (
	ProviderNone             ProviderType = ""
	ProviderOpenAICompatible ProviderType = "openai_compatible"
)

// Config selects the active translation provider and holds its settings.
type Config struct {
	Provider ProviderType
	OpenAI   OpenAIConfig
}

// ErrNoProvider is returned by New when no translation provider is active.
var ErrNoProvider = errors.New("no translation provider configured")

// defaultTimeout bounds a single provider request. Local LLMs can be slow on
// long details texts, so this is generous.
const defaultTimeout = 2 * time.Minute

// New returns the Translator for the active provider in cfg. A nil client
// uses an http.Client with a default timeout.
func New(cfg Config, client *http.Client) (Translator, error) {
	if client == nil {
		client = &http.Client{Timeout: defaultTimeout}
	}

	switch cfg.Provider {
	case ProviderNone:
		return nil, ErrNoProvider
	case ProviderOpenAICompatible:
		return NewOpenAI(cfg.OpenAI, client)
	default:
		return nil, errors.New("unknown translation provider: " + string(cfg.Provider))
	}
}
