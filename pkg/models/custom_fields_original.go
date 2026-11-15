package models

// Reserved scene custom field keys holding the Original metadata of a
// translated scene. Written by the translation feature, read by Tombstones.
// See docs/adr/0001-original-metadata-in-custom-fields.md.
const (
	CustomFieldOriginalTitle   = "original_title"
	CustomFieldOriginalDetails = "original_details"
)
