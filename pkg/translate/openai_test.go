package translate

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeChatServer answers chat completion requests with answer(userMessage)
// and records the received requests.
func fakeChatServer(t *testing.T, answer func(user string) string) (*httptest.Server, *[]*http.Request, *[]chatRequest) {
	t.Helper()
	var reqs []*http.Request
	var bodies []chatRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body chatRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding request: %v", err)
		}
		reqs = append(reqs, r)
		bodies = append(bodies, body)

		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": answer(body.Messages[1].Content)}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)

	return srv, &reqs, &bodies
}

func TestNewOpenAI_Validation(t *testing.T) {
	tests := []struct {
		name string
		cfg  OpenAIConfig
	}{
		{"missing base URL", OpenAIConfig{Model: "m"}},
		{"invalid scheme", OpenAIConfig{BaseURL: "ftp://host/v1", Model: "m"}},
		{"no host", OpenAIConfig{BaseURL: "http://", Model: "m"}},
		{"missing model", OpenAIConfig{BaseURL: "http://host/v1"}},
	}

	for _, tt := range tests {
		_, err := NewOpenAI(tt.cfg, nil)
		assert.Error(t, err, tt.name)
	}
}

func TestOpenAI_Translate(t *testing.T) {
	srv, reqs, bodies := fakeChatServer(t, func(user string) string {
		switch {
		case strings.Contains(user, "中出し"):
			return "Creampie"
		case strings.Contains(user, "SSIS-123"):
			return "SSIS-123 A Summer Day"
		default:
			return "?"
		}
	})

	tr, err := NewOpenAI(OpenAIConfig{BaseURL: srv.URL + "/v1/", Model: "qwen3:14b", APIKey: "secret"}, srv.Client())
	require.NoError(t, err)

	got, err := tr.Translate(context.Background(), []Text{
		{Field: FieldTitle, Value: "SSIS-123 ある夏の日"},
		{Field: FieldDetails, Value: "  "},
		{Field: FieldTag, Value: "中出し"},
	}, "ja", "en")

	require.NoError(t, err)
	assert.Equal(t, []string{"SSIS-123 A Summer Day", "", "Creampie"}, got)

	// the blank text is not sent
	require.Len(t, *reqs, 2)
	r := (*reqs)[0]
	assert.Equal(t, "/v1/chat/completions", r.URL.Path)
	assert.Equal(t, "Bearer secret", r.Header.Get("Authorization"))

	b := (*bodies)[0]
	assert.Equal(t, "qwen3:14b", b.Model)
	assert.False(t, b.Stream)
	require.Len(t, b.Messages, 2)
	assert.Equal(t, DefaultSystemPrompt, b.Messages[0].Content)
	assert.Contains(t, b.Messages[1].Content, "Field: title")
	assert.Contains(t, b.Messages[1].Content, "Source language: ja")
	assert.Contains(t, b.Messages[1].Content, "Target language: en")
	assert.Contains(t, b.Messages[1].Content, "<text>\nSSIS-123 ある夏の日\n</text>")

	assert.NotContains(t, b.Messages[1].Content, "Context:")

	assert.Contains(t, (*bodies)[1].Messages[1].Content, "Field: tag")
}

func TestUserMessage_Context(t *testing.T) {
	got := userMessage(Text{
		Field:   FieldTitle,
		Value:   "ある夏の日",
		Context: "Performers: Yua Mikami\nDetails: ...",
	}, "ja", "en")

	assert.Equal(t, "Field: title\nSource language: ja\nTarget language: en\n\nContext:\nPerformers: Yua Mikami\nDetails: ...\n\n<text>\nある夏の日\n</text>", got)

	// blank context is left out
	got = userMessage(Text{Field: FieldTag, Value: "x", Context: "  \n"}, "ja", "en")
	assert.Equal(t, "Field: tag\nSource language: ja\nTarget language: en\n\n<text>\nx\n</text>", got)
}

func TestOpenAI_CustomPromptAndNoKey(t *testing.T) {
	srv, reqs, bodies := fakeChatServer(t, func(string) string { return "ok" })

	tr, err := NewOpenAI(OpenAIConfig{BaseURL: srv.URL, Model: "m", SystemPrompt: "custom"}, srv.Client())
	require.NoError(t, err)

	_, err = tr.Translate(context.Background(), []Text{{Field: FieldTitle, Value: "x"}}, "ja", "en")
	require.NoError(t, err)

	assert.Empty(t, (*reqs)[0].Header.Get("Authorization"))
	assert.Equal(t, "custom", (*bodies)[0].Messages[0].Content)
}

func TestOpenAI_UnusableAnswers(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"not json", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("<html>")) }},
		{"no choices", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"choices":[]}`)) }},
		{"empty content", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"<think>hmm</think>  "}}]}`))
		}},
	}

	for _, tt := range tests {
		srv := httptest.NewServer(tt.handler)
		tr, err := NewOpenAI(OpenAIConfig{BaseURL: srv.URL, Model: "m"}, srv.Client())
		require.NoError(t, err)

		_, err = tr.Translate(context.Background(), []Text{{Field: FieldTitle, Value: "x"}}, "ja", "en")
		assert.True(t, IsKind(err, ErrorKindResponse), "%s: got %v", tt.name, err)
		srv.Close()
	}
}

func TestOpenAI_AuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	tr, err := NewOpenAI(OpenAIConfig{BaseURL: srv.URL, Model: "m"}, srv.Client())
	require.NoError(t, err)

	_, err = tr.Translate(context.Background(), []Text{{Field: FieldTitle, Value: "x"}}, "ja", "en")
	assert.True(t, IsKind(err, ErrorKindAuth), "got %v", err)
}

func TestOpenAI_Name(t *testing.T) {
	tr, err := NewOpenAI(OpenAIConfig{BaseURL: "http://host/v1", Model: "qwen3:14b"}, nil)
	require.NoError(t, err)
	assert.Equal(t, "openai_compatible:qwen3:14b", tr.Name())
}

func TestCleanAnswer(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"A Summer Day", "A Summer Day"},
		{"  A Summer Day\n", "A Summer Day"},
		{"<think>\nreasoning\n</think>\nA Summer Day", "A Summer Day"},
		{"<text>\nA Summer Day\n</text>", "A Summer Day"},
		{"Translation: A Summer Day", "A Summer Day"},
		{"Here is the translation: A Summer Day", "A Summer Day"},
		{`"A Summer Day"`, "A Summer Day"},
		{"“A Summer Day”", "A Summer Day"},
		{"「A Summer Day」", "A Summer Day"},
		// quotes that do not wrap the whole answer stay
		{`"Hot" and "Cold"`, `"Hot" and "Cold"`},
		{"Line one\n\nLine two", "Line one\n\nLine two"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, cleanAnswer(tt.in), tt.in)
	}
}

func TestNew(t *testing.T) {
	_, err := New(Config{}, nil)
	assert.ErrorIs(t, err, ErrNoProvider)

	_, err = New(Config{Provider: "bogus"}, nil)
	assert.Error(t, err)

	tr, err := New(Config{Provider: ProviderOpenAICompatible, OpenAI: OpenAIConfig{BaseURL: "http://host/v1", Model: "m"}}, nil)
	require.NoError(t, err)
	assert.IsType(t, &OpenAI{}, tr)
}
