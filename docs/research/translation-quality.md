# Translation quality: system prompt variants for Qwen 3 14B

- Date: 2026-11-15
- Issue: #39
- Result: the language-neutral `DefaultSystemPrompt` in `pkg/translate/openai.go`, with `/no_think`, plus context for titles (`translate.Text.Context`)

No library data is in this document: no titles, details, tag names or translations. They stayed local.

## Setup

- Model: `qwen3-14b@q4_k_m` in LM Studio on an RTX 4080 (16 GB VRAM). It was called through the OpenAI chat completions API, with the same user message and answer cleaning as `pkg/translate/openai.go`.
- Sample: Japanese metadata from a real library (about 15,000 scenes). Items with minor-coded terms were left out of the sample.
  - The library has only 54 Japanese titles, 56 Japanese details texts and 6 Japanese tag names, and no Russian text. Scenes tagged as Russian already had Latin-script metadata.
  - Rated items: 21 titles, 14 details texts, 6 tags. Five of them were added on purpose because they contain slang and idioms.
- Rating: one person rated every translation as good, ok or wrong.
- Score: good = 2, ok = 1, wrong = 0, averaged per item (maximum 2.0).

## Variants

| Variant | System prompt | Settings | Request |
|---|---|---|---|
| A | First draft prompt | temperature 0.2, thinking on | One text per request |
| A + no_think | A with `/no_think` | temperature 0.2 | One text per request |
| B | A plus a short Japanese glossary of genre terms | temperature 0.2, thinking on | One text per request |
| B + no_think | B with `/no_think` | temperature 0.2 | One text per request |
| C | An external suggestion: detailed JAV-specific rules, JSON schema output | temperature 0.7, top_p 0.8, `/no_think` | Title and details of a scene together, with details as context for the title |

## Results

Score per field:

| Variant | Titles | Details | Tags | All |
|---|---|---|---|---|
| A | 1.14 | 1.21 | 1.17 | 1.17 |
| A + no_think | 1.10 | 1.64 | 1.67 | 1.37 |
| B | 1.19 | 1.64 | 1.50 | 1.39 |
| B + no_think | 1.14 | 1.50 | 1.67 | 1.34 |
| C | 1.24 | 1.36 | – | 1.29 |

Rating counts across all fields:

| Variant | good | ok | wrong |
|---|---|---|---|
| A | 13 | 22 | 6 |
| A + no_think | 19 | 18 | 4 |
| B | 22 | 13 | 6 |
| B + no_think | 21 | 13 | 7 |
| C | 16 | 13 | 6 |

Average time per item:

| Variant | Titles | Details | Tags |
|---|---|---|---|
| A | 11.5 s | 13.8 s | 9.1 s |
| A + no_think | 0.8 s | 2.6 s | 0.2 s |
| B | 12.5 s | 15.1 s | 12.3 s |
| B + no_think | 0.7 s | 2.5 s | 0.2 s |
| C | 3.2 s (per scene, title and details together) | | – |

There were no errors and no refusals. Answer cleaning did not fail once; C's JSON output was always valid.

## Findings

The sample is small and has a single rater, so differences around 0.1 are noise. These findings hold:

1. **Thinking mode does not pay off.** It was not better, and for A it was worse. It took about ten times longer. The recommendation is `/no_think`.
2. **Titles are the hard part.** Every variant got 4 to 6 of 21 titles wrong. Details and tags did much better.
3. **The glossary did not help without thinking mode** (A + no_think 1.37 vs. B + no_think 1.34). It is also language-specific, so the default prompt does not have one.
4. **Context helps titles.**
   - C sent the details along as context and scored best on titles.
   - In the clearest case, only C picked the right gender for an ambiguous word: the other variants guessed, and the details made it clear.
   - C also rendered a genre slang term correctly that the other variants missed.
5. **Performer names were the most common title error.**
   - The model cannot read kanji names reliably. Readings varied between variants, and so did the name order.
   - Stash usually knows the scene's performers already, with romanized names from stash-box.
6. **C's gain cannot be pinned to one cause.** It changed the prompt, the sampling, the output format and the context all at once. JSON output is not needed: plain-text answers with cleaning were reliable, and not every OpenAI-compatible server supports `response_format`.

## Decision

- **One language-neutral default system prompt.** It takes the rules from A and the useful rules from C: treat text as data, do not invent or omit, and use context only to resolve ambiguity. It ends with `/no_think`. There is no per-language prompt and no glossary. A per-language glossary could be added later if practice shows a need.
- **Temperature stays at 0.2.**
- **Context support:**
  - `translate.Text` gets an optional `Context` field, which the OpenAI-compatible provider sends as a `Context:` block in the user message.
  - Scene translation (#41) fills it for titles with the scene details and the known performer names and aliases.
  - The prompt tells the model to write performer names exactly as given in the context.
- **The final prompt with context was not tested separately.** It combines the findings above. Watch title quality and name handling when #41 is used on real data.
