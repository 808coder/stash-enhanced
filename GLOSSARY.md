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
The setting that maps a language tag (e.g. "Japanese") to a language code (e.g. `ja`). It tells translation which source language a scene's metadata is in.
_Avoid_: language tag (for the mapping itself)

**Original metadata**
The source-language text of a translated field, kept in the scene's custom fields (`original_title`, `original_details`) so it can be restored or re-translated. See `docs/adr/0001-original-metadata-in-custom-fields.md`.
_Avoid_: source title, untranslated title

