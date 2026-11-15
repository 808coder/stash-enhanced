# Original metadata in custom fields

- Status: accepted
- Date: 2026-11-15
- Issues: #40, #41

## Context

Machine translation overwrites a scene's `title` and `details`. The source text has to survive: translations can be wrong, and a better translation provider or prompt should be able to re-translate from the source text later. We also want to know which language and which translation provider produced a translation.

Scenes have no field for an original title. Adding columns means a migration, schema changes in the GraphQL types and the scene model, and more conflicts with upstream merges.

Scenes already have custom fields (`scene_custom_fields`, migration 79): a key/value store per scene that GraphQL can read and that `SceneUpdateInput` and `BulkSceneUpdateInput` can write.

## Decision

Store **Original metadata** in scene custom fields:

| Key | Content |
|---|---|
| `original_title` | Source text of the title, only if the title was translated |
| `original_details` | Source text of the details, only if the details were translated |
| `translation_source_lang` | Language code of the source text, e.g. `ja` |
| `translation_provider` | Translation provider and model that produced the translation |
| `translated_at` | When the translation was made |

The translated text goes into the regular `title` and `details` fields, so everything that reads them (lists, search, DLNA, plugins) sees the translation without changes.

Tags are not covered: a translated tag keeps its source-language name as an alias.

## Consequences

- No migration and no schema change for scenes.
- "Restore original" writes `original_*` back and removes the five keys.
- The keys are visible and editable in the custom fields UI like any other custom field. Editing them by hand changes what a re-translation starts from.
- Custom field keys are plain strings. The keys above are reserved by convention, not enforced.
