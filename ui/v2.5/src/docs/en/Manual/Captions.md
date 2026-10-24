# Captions

Stash supports captioning with SRT and VTT files.

Captions will only be detected if they are located in the same folder as the corresponding scene file.

Ensure the caption files follow these naming conventions:

## Scene

- {scene_file_name}.{language_code}.{ext}
- {scene_file_name}.{ext}

Where `{language_code}` is defined by the [ISO-6399-1](https://en.wikipedia.org/wiki/List_of_ISO_639-1_codes) (2 letters) standard and `{ext}` is the file extension. Captions files without a language code will be labeled as Unknown in the video player but will work fine.

Scenes with captions can be filtered with the `captions` criterion.

## Captions tab

Scenes with captions have a **Captions** tab on the scene details page, after the Markers tab. It lists every cue of the selected caption file with its start time and text.

- If the scene has more than one caption file, use the dropdown to choose the language and type. The first caption is selected by default.
- Use the filter field to show only cues containing the entered text (case-insensitive).
- Click a timecode to seek the player to the start of that cue.
- The cue at the current playback position is highlighted while the tab is open.

Captions are only loaded when the tab is first opened.

> **⚠️ Note:** If the caption file was added after the scene was initially added during scan, you will need to run a Selective scan task for it to show up.
