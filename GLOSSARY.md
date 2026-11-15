# Glossary

## Files

**Path**
The full, absolute location of a file or folder on disk, including all parent directories and the name itself (e.g. `/media/videos/clip.mp4`).
_Avoid_: location, file path (when the name alone is meant)

**Filename**
The last segment of a file's Path, including its extension (e.g. `clip.mp4`). Called `basename` in the GraphQL schema.
_Avoid_: basename (in user-facing text), name, title

## Translation

**Translation provider**
A service that machine-translates metadata text, such as an OpenAI-compatible chat API (Ollama, LM Studio) or Azure Translator. Exactly one is active at a time.
_Avoid_: translator (for the service; `Translator` is the Go interface), engine, backend

**Language mapping**
The setting that maps a language tag (e.g. "Japanese") to a language code (e.g. `ja`). It tells translation which source language a scene's metadata is in. Tags have no language tag: their source language is detected from the script of the name, and only languages in the mapping count.
_Avoid_: language tag (for the mapping itself)

**Original metadata**
The source-language text of a translated field, kept in the scene's custom fields (`original_title`, `original_details`) so it can be restored or re-translated. See `docs/adr/0001-original-metadata-in-custom-fields.md`.
_Avoid_: source title, untranslated title

**Tag collision**
A tag's translation matches the name or alias of another tag, case-insensitively. With **Merge on collision** the tag is merged into the other tag; otherwise it is skipped.
_Avoid_: duplicate tag, conflict

