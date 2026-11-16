package models

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"time"
)

// TombstoneOrigin records which operation removed the scene that a
// Tombstone describes.
type TombstoneOrigin string

const (
	// TombstoneOriginManual is a scene destroyed by the user with "delete file".
	TombstoneOriginManual TombstoneOrigin = "MANUAL"
	// TombstoneOriginClean is a scene removed by the Clean task because its
	// file was missing from disk.
	TombstoneOriginClean TombstoneOrigin = "CLEAN"
)

var AllTombstoneOrigin = []TombstoneOrigin{
	TombstoneOriginManual,
	TombstoneOriginClean,
}

func (e TombstoneOrigin) IsValid() bool {
	switch e {
	case TombstoneOriginManual, TombstoneOriginClean:
		return true
	}
	return false
}

func (e TombstoneOrigin) String() string {
	return string(e)
}

func (e *TombstoneOrigin) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = TombstoneOrigin(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid TombstoneOrigin", str)
	}
	return nil
}

func (e TombstoneOrigin) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

// TombstoneMatchKey names the filter key a Tombstone matched on in a
// Tombstone query.
type TombstoneMatchKey string

const (
	TombstoneMatchKeyURL      TombstoneMatchKey = "URL"
	TombstoneMatchKeyStashID  TombstoneMatchKey = "STASH_ID"
	TombstoneMatchKeyOshash   TombstoneMatchKey = "OSHASH"
	TombstoneMatchKeyMD5      TombstoneMatchKey = "MD5"
	TombstoneMatchKeyPhash    TombstoneMatchKey = "PHASH"
	TombstoneMatchKeyCode     TombstoneMatchKey = "CODE"
	TombstoneMatchKeyTitle    TombstoneMatchKey = "TITLE"
	TombstoneMatchKeyBasename TombstoneMatchKey = "BASENAME"
)

var AllTombstoneMatchKey = []TombstoneMatchKey{
	TombstoneMatchKeyURL,
	TombstoneMatchKeyStashID,
	TombstoneMatchKeyOshash,
	TombstoneMatchKeyMD5,
	TombstoneMatchKeyPhash,
	TombstoneMatchKeyCode,
	TombstoneMatchKeyTitle,
	TombstoneMatchKeyBasename,
}

func (e TombstoneMatchKey) IsValid() bool {
	for _, k := range AllTombstoneMatchKey {
		if e == k {
			return true
		}
	}
	return false
}

func (e TombstoneMatchKey) String() string {
	return string(e)
}

func (e *TombstoneMatchKey) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = TombstoneMatchKey(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid TombstoneMatchKey", str)
	}
	return nil
}

func (e TombstoneMatchKey) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

// TombstoneStashID is a frozen stash ID (endpoint and id) of a deleted
// scene, or of its studio or performers.
type TombstoneStashID struct {
	Endpoint string `json:"endpoint"`
	StashID  string `json:"stash_id"`
}

// TombstoneEntity is a frozen related entity (studio or performer):
// its name as plain text plus its stash IDs.
type TombstoneEntity struct {
	Name     string             `json:"name"`
	StashIDs []TombstoneStashID `json:"stash_ids"`
}

// TombstonePerformer is a frozen performer. Rows stored before gender and
// the former id were recorded decode with ID 0 and a nil Gender.
type TombstonePerformer struct {
	// ID is the id the performer had when the Tombstone was recorded. The
	// performer may since have been deleted and the id reused.
	ID       int                `json:"id,omitempty"`
	Name     string             `json:"name"`
	Gender   *GenderEnum        `json:"gender,omitempty"`
	StashIDs []TombstoneStashID `json:"stash_ids"`
}

// Tombstone is the record left behind for one deleted scene: a frozen
// snapshot of its identifying and descriptive data plus its files.
// Everything except Reason is immutable once stored.
type Tombstone struct {
	ID        int             `json:"id"`
	DeletedAt time.Time       `json:"deleted_at"`
	Origin    TombstoneOrigin `json:"origin"`
	// Reason is optional free text; empty means none.
	Reason string `json:"reason"`
	// SceneID is the id the scene had before it was deleted.
	SceneID int `json:"scene_id"`

	Title           string             `json:"title"`
	Code            string             `json:"code"`
	Date            *Date              `json:"date"`
	URLs            []string           `json:"urls"`
	StashIDs        []TombstoneStashID `json:"stash_ids"`
	OriginalTitle   string             `json:"original_title"`
	OriginalDetails string             `json:"original_details"`
	Details         string             `json:"details"`
	// Rating is expressed as 1-100.
	Rating    *int `json:"rating"`
	OCounter  int  `json:"o_counter"`
	PlayCount int  `json:"play_count"`
	// SceneCreatedAt is when the scene was added to the library.
	SceneCreatedAt time.Time `json:"scene_created_at"`

	Studio     *TombstoneEntity     `json:"studio"`
	Performers []TombstonePerformer `json:"performers"`
	Tags       []string             `json:"tags"`

	Files []*TombstoneFile `json:"files"`
}

// TombstoneFile is one of the files listed under a Tombstone.
type TombstoneFile struct {
	ID          int    `json:"id"`
	TombstoneID int    `json:"tombstone_id"`
	Basename    string `json:"basename"`
	Path        string `json:"path"`
	Size        int64  `json:"size"`
	// Duration in seconds.
	Duration   float64 `json:"duration"`
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	VideoCodec string  `json:"video_codec"`
	BitRate    int64   `json:"bit_rate"`
	Oshash     string  `json:"oshash"`
	MD5        string  `json:"md5"`
	// Phash is nil when the file had no phash.
	Phash *int64 `json:"phash"`
}

// TombstonePhashInput matches Tombstone files whose phash is less than
// Distance apart from Value (exclusive, like the scene phash_distance
// filter). Distance 0 matches the exact phash.
type TombstonePhashInput struct {
	Value    int64
	Distance int
}

// TombstoneFilter selects Tombstones. Every non-empty field except Origin is
// one key; a Tombstone matches when it matches at least one key (OR). A
// filter without keys matches every Tombstone. Origin is not a key: it
// restricts the result (AND) and is never reported as a matched key.
type TombstoneFilter struct {
	// Only Tombstones of this Origin. Nil matches every Origin.
	Origin *TombstoneOrigin
	// Exact match against the scene URLs.
	URL string
	// Exact match against the id of the scene stash IDs (any endpoint).
	StashID string
	// Exact match against a file oshash.
	Oshash string
	// Exact match against a file md5.
	MD5 string
	// Exact match against the studio code.
	Code string
	// Match against a file phash within a maximum distance.
	Phash *TombstonePhashInput
	// Case-insensitive substring of the title or the original title.
	Title string
	// Case-insensitive substring of a file Filename stem (see
	// NormaliseTombstoneStem). The term is searched as given; the API layer
	// strips video extensions and resolution tokens from it beforehand.
	Basename string
}

// tombstoneResolutionToken matches a trailing resolution token of a release
// name, such as ".1080p", ".720P" or ".4k".
var tombstoneResolutionToken = regexp.MustCompile(`(?i)\.(\d{3,4}p|4k)$`)

// NormaliseTombstoneStem strips a trailing resolution token (".<3-4
// digits>p" or ".4k", case-insensitive) from a Filename stem, so that the
// same release in another quality matches. It is applied to stored stems
// and to search terms alike. A stem that consists of the token only is
// returned unchanged.
func NormaliseTombstoneStem(stem string) string {
	if ret := tombstoneResolutionToken.ReplaceAllString(stem, ""); ret != "" {
		return ret
	}
	return stem
}

// IsEmpty reports whether the filter has no keys. Origin is not a key.
func (f *TombstoneFilter) IsEmpty() bool {
	return f == nil || (f.URL == "" && f.StashID == "" && f.Oshash == "" && f.MD5 == "" &&
		f.Code == "" && f.Phash == nil && f.Title == "" && f.Basename == "")
}

// TombstoneMatch is one Tombstone query result with the keys it matched on.
// MatchedKeys is empty when the query had no filter keys.
type TombstoneMatch struct {
	Tombstone   *Tombstone          `json:"tombstone"`
	MatchedKeys []TombstoneMatchKey `json:"matched_keys"`
}

// TombstoneSort names the sort orders of a Tombstone query. The file sorts
// use the first file of a Tombstone (lowest id).
const (
	TombstoneSortDeletedAt  = "deleted_at"
	TombstoneSortTitle      = "title"
	TombstoneSortOrigin     = "origin"
	TombstoneSortReason     = "reason"
	TombstoneSortSize       = "size"
	TombstoneSortResolution = "resolution"
	TombstoneSortBasename   = "basename"
	TombstoneSortPhash      = "phash"
)

// TombstoneQueryResult is one page of Tombstone matches in the requested
// sort order and the total number of matching Tombstones. ManualCount and
// CleanCount are the totals per Origin of the Tombstones matching the filter
// keys, ignoring the filter's Origin.
type TombstoneQueryResult struct {
	Count       int               `json:"count"`
	ManualCount int               `json:"manual_count"`
	CleanCount  int               `json:"clean_count"`
	Matches     []*TombstoneMatch `json:"matches"`
}
