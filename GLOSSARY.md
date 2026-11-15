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


## Tombstones

**Tombstone**
The record left behind for one scene that was deleted together with its file, by the user ("delete file") or by the Clean task. A frozen snapshot of the scene's identifying and descriptive data; related studio, performers and tags are kept as plain text. Only its reason can be edited. Stored in `fork_tombstones`, see `docs/adr/0002-fork-schema-outside-upstream-migrations.md`.
_Avoid_: deleted scene, history entry, blacklist

**Tombstone file**
One of the files listed under a Tombstone: its Filename, Path, quality data and fingerprints (oshash, md5, phash). Stored in `fork_tombstone_files`.
_Avoid_: deleted file

**Origin**
Why a Tombstone exists: `manual` (the user deleted the scene with its file) or `clean` (the Clean task removed the scene because its file was missing).
_Avoid_: source, type
