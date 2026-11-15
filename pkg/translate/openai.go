package translate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// DefaultSystemPrompt is used when no system prompt is configured.
//
// This is a first draft. #39 compares translation quality on real library data
// and replaces it with the recommended prompt.
const DefaultSystemPrompt = `You translate metadata of adult video scenes.

The user message gives the field type (title, details or tag), the source language, the target language and the text between <text> and </text>.

Rules:
- Output only the translation. No quotes, notes, explanations or the <text> markers.
- Keep product codes (e.g. SSIS-123), words already written in Latin letters, numbers and URLs unchanged.
- Romanize personal names (Hepburn for Japanese), given name first.
- Keep the meaning and tone, including explicit terms. Do not censor or soften.
- title: a natural, concise title.
- details: keep paragraphs and line breaks.
- tag: a short tag name of one to three words in title case.`

// OpenAIConfig configures a provider that speaks the OpenAI chat completions
// API, such as Ollama, LM Studio, vLLM or a cloud service.
type OpenAIConfig struct {
	// BaseURL is the API root including the version, e.g.
	// "http://192.168.1.50:11434/v1". "/chat/completions" is appended.
	BaseURL string
	Model   string
	// APIKey is optional; local servers usually need none.
	APIKey string
	// SystemPrompt defaults to DefaultSystemPrompt when empty.
	SystemPrompt string
}

// OpenAI is a Translator using an OpenAI-compatible chat completions API. It
// sends one request per text.
type OpenAI struct {
	cfg      OpenAIConfig
	endpoint string
	client   *http.Client
	retry    retryPolicy
}

// NewOpenAI returns an OpenAI-compatible Translator. It fails if the base URL
// or model is missing or invalid.
func NewOpenAI(cfg OpenAIConfig, client *http.Client) (*OpenAI, error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		return nil, errors.New("base URL is required")
	}
	u, err := url.Parse(base)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("invalid base URL: %q", cfg.BaseURL)
	}

	cfg.Model = strings.TrimSpace(cfg.Model)
	if cfg.Model == "" {
		return nil, errors.New("model is required")
	}

	if strings.TrimSpace(cfg.SystemPrompt) == "" {
		cfg.SystemPrompt = DefaultSystemPrompt
	}

	if client == nil {
		client = &http.Client{Timeout: defaultTimeout}
	}

	return &OpenAI{
		cfg:      cfg,
		endpoint: base + "/chat/completions",
		client:   client,
		retry:    defaultRetryPolicy(),
	}, nil
}

func (t *OpenAI) Name() string {
	return string(ProviderOpenAICompatible) + ":" + t.cfg.Model
}

func (t *OpenAI) Translate(ctx context.Context, texts []Text, from, to string) ([]string, error) {
	ret := make([]string, len(texts))
	for i, text := range texts {
		if strings.TrimSpace(text.Value) == "" {
			continue
		}

		translated, err := t.translateOne(ctx, text, from, to)
		if err != nil {
			return nil, err
		}
		ret[i] = translated
	}

	return ret, nil
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	Stream      bool          `json:"stream"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

func (t *OpenAI) translateOne(ctx context.Context, text Text, from, to string) (string, error) {
	payload, err := json.Marshal(chatRequest{
		Model: t.cfg.Model,
		Messages: []chatMessage{
			{Role: "system", Content: t.cfg.SystemPrompt},
			{Role: "user", Content: userMessage(text, from, to)},
		},
		Temperature: 0.2,
	})
	if err != nil {
		return "", newError(ErrorKindResponse, "encoding request: %w", err)
	}

	body, err := t.retry.do(ctx, t.client, func(ctx context.Context) (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.endpoint, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		if t.cfg.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+t.cfg.APIKey)
		}
		return req, nil
	})
	if err != nil {
		return "", err
	}

	var resp chatResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", newError(ErrorKindResponse, "decoding response: %w: %s", err, snippet(body))
	}
	if len(resp.Choices) == 0 {
		return "", newError(ErrorKindResponse, "response has no choices: %s", snippet(body))
	}

	translated := cleanAnswer(resp.Choices[0].Message.Content)
	if translated == "" {
		return "", newError(ErrorKindResponse, "empty translation")
	}

	return translated, nil
}

func userMessage(text Text, from, to string) string {
	return fmt.Sprintf("Field: %s\nSource language: %s\nTarget language: %s\n\n<text>\n%s\n</text>", text.Field, from, to, text.Value)
}

var (
	thinkBlock    = regexp.MustCompile(`(?s)<think>.*?</think>`)
	answerPrefix  = regexp.MustCompile(`(?i)^(here is the translation|translation|translated text)\s*:\s*`)
	quotePairs    = [][2]string{{`"`, `"`}, {"“", "”"}, {"「", "」"}, {"'", "'"}}
	textMarkerTag = strings.NewReplacer("<text>", "", "</text>", "")
)

// cleanAnswer strips what chat models commonly wrap around a translation:
// reasoning blocks, echoed <text> markers, a "Translation:" prefix and
// surrounding quotes.
func cleanAnswer(s string) string {
	s = thinkBlock.ReplaceAllString(s, "")
	s = textMarkerTag.Replace(s)
	s = strings.TrimSpace(s)
	s = answerPrefix.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)

	for _, q := range quotePairs {
		if len(s) >= len(q[0])+len(q[1]) && strings.HasPrefix(s, q[0]) && strings.HasSuffix(s, q[1]) {
			inner := s[len(q[0]) : len(s)-len(q[1])]
			// only strip if the quotes wrap the whole answer
			if !strings.Contains(inner, q[1]) {
				s = strings.TrimSpace(inner)
			}
			break
		}
	}

	return s
}
