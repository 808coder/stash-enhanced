package models

// Reserved scene custom field keys holding the Original metadata of a
// translated scene. Written by the translation feature, read by Tombstones.
// See docs/adr/0001-original-metadata-in-custom-fields.md.
const (
	CustomFieldOriginalTitle   = "original_title"
	CustomFieldOriginalDetails = "original_details"
)

// CustomFieldString returns the string value of a custom field, or an empty
// string when the key is absent or not a string.
func CustomFieldString(fields map[string]interface{}, key string) string {
	s, _ := fields[key].(string)
	return s
}
