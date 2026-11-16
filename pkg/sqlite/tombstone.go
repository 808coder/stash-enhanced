package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/jmoiron/sqlx"
	"gopkg.in/guregu/null.v4"
	"gopkg.in/guregu/null.v4/zero"

	"github.com/stashapp/stash/pkg/models"
)

const (
	tombstoneTable     = "fork_tombstones"
	tombstoneFileTable = "fork_tombstone_files"
)

type tombstoneRow struct {
	ID              int           `db:"id" goqu:"skipinsert"`
	DeletedAt       UTCTimestamp  `db:"deleted_at"`
	Origin          string        `db:"origin"`
	Reason          zero.String   `db:"reason"`
	SceneID         null.Int      `db:"scene_id"`
	Title           zero.String   `db:"title"`
	Code            zero.String   `db:"code"`
	Date            NullDate      `db:"date"`
	DatePrecision   null.Int      `db:"date_precision"`
	URLs            string        `db:"urls"`
	StashIDs        string        `db:"stash_ids"`
	OriginalTitle   zero.String   `db:"original_title"`
	OriginalDetails zero.String   `db:"original_details"`
	Details         zero.String   `db:"details"`
	Rating          null.Int      `db:"rating"`
	OCounter        int           `db:"o_counter"`
	PlayCount       int           `db:"play_count"`
	SceneCreatedAt  NullTimestamp `db:"scene_created_at"`
	Studio          zero.String   `db:"studio"`
	Performers      string        `db:"performers"`
	Tags            string        `db:"tags"`
}

func encodeJSONList[T any](v []T) (string, error) {
	if v == nil {
		v = []T{}
	}
	b, err := json.Marshal(v)
	return string(b), err
}

func (r *tombstoneRow) fromTombstone(o models.Tombstone) error {
	var err error
	r.ID = o.ID
	r.DeletedAt = UTCTimestamp{Timestamp{Timestamp: o.DeletedAt}}
	r.Origin = string(o.Origin)
	r.Reason = zero.StringFrom(o.Reason)
	r.SceneID = null.NewInt(int64(o.SceneID), o.SceneID != 0)
	r.Title = zero.StringFrom(o.Title)
	r.Code = zero.StringFrom(o.Code)
	r.Date = NullDateFromDatePtr(o.Date)
	r.DatePrecision = datePrecisionFromDatePtr(o.Date)
	r.OriginalTitle = zero.StringFrom(o.OriginalTitle)
	r.OriginalDetails = zero.StringFrom(o.OriginalDetails)
	r.Details = zero.StringFrom(o.Details)
	r.Rating = intFromPtr(o.Rating)
	r.OCounter = o.OCounter
	r.PlayCount = o.PlayCount
	if !o.SceneCreatedAt.IsZero() {
		r.SceneCreatedAt = NullTimestamp{Timestamp: o.SceneCreatedAt.UTC(), Valid: true}
	}

	if r.URLs, err = encodeJSONList(o.URLs); err != nil {
		return err
	}
	if r.StashIDs, err = encodeJSONList(o.StashIDs); err != nil {
		return err
	}
	if r.Performers, err = encodeJSONList(o.Performers); err != nil {
		return err
	}
	if r.Tags, err = encodeJSONList(o.Tags); err != nil {
		return err
	}
	if o.Studio != nil {
		b, err := json.Marshal(o.Studio)
		if err != nil {
			return err
		}
		r.Studio = zero.StringFrom(string(b))
	}

	return nil
}

func (r *tombstoneRow) resolve() (*models.Tombstone, error) {
	ret := &models.Tombstone{
		ID:              r.ID,
		DeletedAt:       r.DeletedAt.Timestamp.Timestamp.UTC(),
		Origin:          models.TombstoneOrigin(r.Origin),
		Reason:          r.Reason.String,
		SceneID:         int(r.SceneID.Int64),
		Title:           r.Title.String,
		Code:            r.Code.String,
		Date:            r.Date.DatePtr(r.DatePrecision),
		OriginalTitle:   r.OriginalTitle.String,
		OriginalDetails: r.OriginalDetails.String,
		Details:         r.Details.String,
		Rating:          nullIntPtr(r.Rating),
		OCounter:        r.OCounter,
		PlayCount:       r.PlayCount,
	}
	if r.SceneCreatedAt.Valid {
		ret.SceneCreatedAt = r.SceneCreatedAt.Timestamp.UTC()
	}

	for _, v := range []struct {
		s    string
		dest interface{}
	}{
		{r.URLs, &ret.URLs},
		{r.StashIDs, &ret.StashIDs},
		{r.Performers, &ret.Performers},
		{r.Tags, &ret.Tags},
	} {
		if v.s == "" || v.s == "[]" {
			continue
		}
		if err := json.Unmarshal([]byte(v.s), v.dest); err != nil {
			return nil, fmt.Errorf("decoding tombstone %d: %w", r.ID, err)
		}
	}

	if r.Studio.Valid && r.Studio.String != "" {
		ret.Studio = &models.TombstoneStudio{}
		if err := json.Unmarshal([]byte(r.Studio.String), ret.Studio); err != nil {
			return nil, fmt.Errorf("decoding tombstone %d studio: %w", r.ID, err)
		}
	}

	return ret, nil
}

type tombstoneFileRow struct {
	ID           int         `db:"id" goqu:"skipinsert"`
	TombstoneID  int         `db:"tombstone_id"`
	Basename     string      `db:"basename"`
	BasenameStem string      `db:"basename_stem"`
	Path         string      `db:"path"`
	Size         int64       `db:"size"`
	Duration     float64     `db:"duration"`
	Width        int         `db:"width"`
	Height       int         `db:"height"`
	VideoCodec   string      `db:"video_codec"`
	BitRate      int64       `db:"bit_rate"`
	Oshash       zero.String `db:"oshash"`
	MD5          zero.String `db:"md5"`
	Phash        null.Int    `db:"phash"`
}

// basenameStem returns the Filename without its extension and without a
// trailing resolution token (see models.NormaliseTombstoneStem).
func basenameStem(basename string) string {
	return models.NormaliseTombstoneStem(strings.TrimSuffix(basename, filepath.Ext(basename)))
}

func (r *tombstoneFileRow) fromTombstoneFile(o models.TombstoneFile) {
	r.ID = o.ID
	r.TombstoneID = o.TombstoneID
	r.Basename = o.Basename
	r.BasenameStem = basenameStem(o.Basename)
	r.Path = o.Path
	r.Size = o.Size
	r.Duration = o.Duration
	r.Width = o.Width
	r.Height = o.Height
	r.VideoCodec = o.VideoCodec
	r.BitRate = o.BitRate
	r.Oshash = zero.StringFrom(o.Oshash)
	r.MD5 = zero.StringFrom(o.MD5)
	if o.Phash != nil {
		r.Phash = null.IntFrom(*o.Phash)
	}
}

func (r *tombstoneFileRow) resolve() *models.TombstoneFile {
	ret := &models.TombstoneFile{
		ID:          r.ID,
		TombstoneID: r.TombstoneID,
		Basename:    r.Basename,
		Path:        r.Path,
		Size:        r.Size,
		Duration:    r.Duration,
		Width:       r.Width,
		Height:      r.Height,
		VideoCodec:  r.VideoCodec,
		BitRate:     r.BitRate,
		Oshash:      r.Oshash.String,
		MD5:         r.MD5.String,
	}
	if r.Phash.Valid {
		v := r.Phash.Int64
		ret.Phash = &v
	}
	return ret
}

// TombstoneStore stores Tombstones in the fork-owned tables. See ADR 0002.
type TombstoneStore struct {
	tableMgr     *table
	fileTableMgr *table
}

func NewTombstoneStore() *TombstoneStore {
	return &TombstoneStore{
		tableMgr:     tombstoneTableMgr,
		fileTableMgr: tombstoneFileTableMgr,
	}
}

func (qb *TombstoneStore) Create(ctx context.Context, newTombstone *models.Tombstone) error {
	if newTombstone.DeletedAt.IsZero() {
		newTombstone.DeletedAt = time.Now()
	}

	var r tombstoneRow
	if err := r.fromTombstone(*newTombstone); err != nil {
		return err
	}

	id, err := qb.tableMgr.insertID(ctx, r)
	if err != nil {
		return fmt.Errorf("inserting tombstone: %w", err)
	}

	for _, f := range newTombstone.Files {
		f.TombstoneID = id
		var fr tombstoneFileRow
		fr.fromTombstoneFile(*f)
		fid, err := qb.fileTableMgr.insertID(ctx, fr)
		if err != nil {
			return fmt.Errorf("inserting tombstone file: %w", err)
		}
		f.ID = fid
	}

	newTombstone.ID = id
	newTombstone.DeletedAt = r.DeletedAt.Timestamp.Timestamp.UTC().Truncate(time.Second)

	return nil
}

func (qb *TombstoneStore) UpdateReason(ctx context.Context, id int, reason string) error {
	exists, err := qb.tableMgr.idExists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return &NotFoundError{ID: id, Table: tombstoneTable}
	}

	return qb.tableMgr.updateByID(ctx, id, goqu.Record{"reason": zero.StringFrom(reason)})
}

func (qb *TombstoneStore) Destroy(ctx context.Context, id int) error {
	return qb.tableMgr.destroyExisting(ctx, []int{id})
}

// returns nil, nil if not found
func (qb *TombstoneStore) Find(ctx context.Context, id int) (*models.Tombstone, error) {
	ret, err := qb.findMany(ctx, []int{id})
	if err != nil {
		return nil, err
	}
	if len(ret) == 0 {
		return nil, nil
	}
	return ret[0], nil
}

// findMany returns the Tombstones with the given ids, with their files, in
// the order of ids. Missing ids are skipped.
func (qb *TombstoneStore) findMany(ctx context.Context, ids []int) ([]*models.Tombstone, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	q := dialect.From(qb.tableMgr.table).Select(qb.tableMgr.table.All()).
		Prepared(true).Where(qb.tableMgr.idColumn.In(ids))

	byID := make(map[int]*models.Tombstone, len(ids))
	if err := queryFunc(ctx, q, false, func(rows *sqlx.Rows) error {
		var r tombstoneRow
		if err := rows.StructScan(&r); err != nil {
			return err
		}
		t, err := r.resolve()
		if err != nil {
			return err
		}
		byID[t.ID] = t
		return nil
	}); err != nil {
		return nil, fmt.Errorf("finding tombstones: %w", err)
	}

	fq := dialect.From(qb.fileTableMgr.table).Select(qb.fileTableMgr.table.All()).
		Prepared(true).Where(qb.fileTableMgr.table.Col("tombstone_id").In(ids)).
		Order(qb.fileTableMgr.idColumn.Asc())
	if err := queryFunc(ctx, fq, false, func(rows *sqlx.Rows) error {
		var r tombstoneFileRow
		if err := rows.StructScan(&r); err != nil {
			return err
		}
		if t := byID[r.TombstoneID]; t != nil {
			t.Files = append(t.Files, r.resolve())
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("finding tombstone files: %w", err)
	}

	ret := make([]*models.Tombstone, 0, len(byID))
	for _, id := range ids {
		if t := byID[id]; t != nil {
			ret = append(ret, t)
		}
	}
	return ret, nil
}

func (qb *TombstoneStore) FindByFileFingerprints(ctx context.Context, oshash string, md5 string) ([]*models.Tombstone, error) {
	var where []string
	var args []interface{}
	if oshash != "" {
		where = append(where, "oshash = ?")
		args = append(args, oshash)
	}
	if md5 != "" {
		where = append(where, "md5 = ?")
		args = append(args, md5)
	}
	if len(where) == 0 {
		return nil, nil
	}

	query := "SELECT DISTINCT tombstone_id FROM " + tombstoneFileTable +
		" WHERE " + strings.Join(where, " OR ") + " ORDER BY tombstone_id"

	var ids []int
	if err := dbWrapper.Select(ctx, &ids, query, args...); err != nil {
		return nil, fmt.Errorf("finding tombstones by fingerprint: %w", err)
	}

	return qb.findMany(ctx, ids)
}

// tombstoneCondition is the SQL condition for one filter key, evaluated
// against fork_tombstones aliased as t.
type tombstoneCondition struct {
	key  models.TombstoneMatchKey
	sql  string
	args []interface{}
}

func tombstoneFileCondition(cond string) string {
	return "EXISTS (SELECT 1 FROM " + tombstoneFileTable + " f WHERE f.tombstone_id = t.id AND " + cond + ")"
}

func containsPattern(s string) string {
	return "%" + escapeLike(s) + "%"
}

// tombstoneConditions returns one condition per non-empty filter key, in
// the order of models.AllTombstoneMatchKey.
func tombstoneConditions(f *models.TombstoneFilter) []tombstoneCondition {
	if f.IsEmpty() {
		return nil
	}

	var ret []tombstoneCondition
	add := func(key models.TombstoneMatchKey, sql string, args ...interface{}) {
		ret = append(ret, tombstoneCondition{key: key, sql: sql, args: args})
	}

	if f.URL != "" {
		add(models.TombstoneMatchKeyURL, "EXISTS (SELECT 1 FROM json_each(t.urls) j WHERE j.value = ?)", f.URL)
	}
	if f.StashID != "" {
		add(models.TombstoneMatchKeyStashID, "EXISTS (SELECT 1 FROM json_each(t.stash_ids) j WHERE json_extract(j.value, '$.stash_id') = ?)", f.StashID)
	}
	if f.Oshash != "" {
		add(models.TombstoneMatchKeyOshash, tombstoneFileCondition("f.oshash = ?"), f.Oshash)
	}
	if f.MD5 != "" {
		add(models.TombstoneMatchKeyMD5, tombstoneFileCondition("f.md5 = ?"), f.MD5)
	}
	if f.Phash != nil {
		// like findScenes' phash_distance criterion: a distance d > 0 matches
		// phashes less than d apart, distance 0 only the exact phash
		limit := max(f.Phash.Distance, 1)
		add(models.TombstoneMatchKeyPhash,
			tombstoneFileCondition("typeof(f.phash) = 'integer' AND phash_distance(f.phash, ?) < ?"),
			f.Phash.Value, limit)
	}
	if f.Code != "" {
		add(models.TombstoneMatchKeyCode, "t.code = ?", f.Code)
	}
	// the text filters fold case with lower_unicode, since LIKE folds ASCII
	// only
	if f.Title != "" {
		p := containsPattern(lowerUnicode(f.Title))
		add(models.TombstoneMatchKeyTitle,
			`(lower_unicode(t.title) LIKE ? ESCAPE '\' OR lower_unicode(t.original_title) LIKE ? ESCAPE '\')`, p, p)
	}
	if f.Basename != "" {
		// the term is searched as given; the API layer normalises it
		p := containsPattern(lowerUnicode(f.Basename))
		add(models.TombstoneMatchKeyBasename,
			tombstoneFileCondition(`lower_unicode(f.basename_stem) LIKE ? ESCAPE '\'`), p)
	}

	return ret
}

// tombstoneFirstFile selects a column of the first file (lowest id) of the
// Tombstone aliased as t.
func tombstoneFirstFile(col string) string {
	return "(SELECT f." + col + " FROM " + tombstoneFileTable + " f WHERE f.tombstone_id = t.id ORDER BY f.id LIMIT 1)"
}

// tombstoneSorts maps the sort keys of a Tombstone query to SQL expressions
// over fork_tombstones aliased as t. An expression yields NULL when the
// Tombstone has no value for the key.
var tombstoneSorts = map[models.TombstoneSort]string{
	models.TombstoneSortDeletedAt:  "t.deleted_at",
	models.TombstoneSortTitle:      "COALESCE(NULLIF(t.title, ''), NULLIF(t.original_title, '')) COLLATE NATURAL_CI",
	models.TombstoneSortOrigin:     "t.origin",
	models.TombstoneSortReason:     "NULLIF(t.reason, '') COLLATE NOCASE",
	models.TombstoneSortSize:       tombstoneFirstFile("size"),
	models.TombstoneSortResolution: tombstoneFirstFile("height"),
	models.TombstoneSortBasename:   "NULLIF(" + tombstoneFirstFile("basename") + ", '') COLLATE NATURAL_CI",
	models.TombstoneSortPhash:      tombstoneFirstFile("phash"),
}

// tombstoneOrderBy returns the ORDER BY clause for findFilter. Without a
// sort it is newest first. Tombstones without a value for the sort key come
// last in either direction; ties are broken newest first. The title and
// Filename sorts use NATURAL_CI, which never reports equal values, so their
// ties follow the sort direction instead.
func tombstoneOrderBy(findFilter *models.FindFilterType) (string, error) {
	const newestFirst = "t.deleted_at DESC, t.id DESC"

	sort := findFilter.GetSort("")
	if sort == "" {
		return " ORDER BY " + newestFirst, nil
	}

	expr, ok := tombstoneSorts[models.TombstoneSort(sort)]
	if !ok {
		return "", fmt.Errorf("invalid tombstone sort %q", sort)
	}

	// GetDirection only ever returns ASC or DESC
	dir := findFilter.GetDirection()

	terms := []string{"(" + expr + ") IS NULL"}
	if models.TombstoneSort(sort) == models.TombstoneSortPhash {
		// phashes are stored signed; order them as shown, by their hex string
		terms = append(terms, "("+expr+") < 0 "+dir)
	}
	terms = append(terms, expr+" "+dir, newestFirst)

	return " ORDER BY " + strings.Join(terms, ", "), nil
}

func (qb *TombstoneStore) Query(ctx context.Context, filter *models.TombstoneFilter, findFilter *models.FindFilterType) (*models.TombstoneQueryResult, error) {
	if findFilter == nil {
		findFilter = &models.FindFilterType{}
	}

	orderBy, err := tombstoneOrderBy(findFilter)
	if err != nil {
		return nil, err
	}

	conds := tombstoneConditions(filter)

	// the keys are ORed; the origin restricts that result
	keysWhere := ""
	var keysArgs []interface{}
	if len(conds) > 0 {
		parts := make([]string, len(conds))
		for i, c := range conds {
			parts[i] = c.sql
			keysArgs = append(keysArgs, c.args...)
		}
		keysWhere = "(" + strings.Join(parts, " OR ") + ")"
	}

	whereParts := []string{}
	whereArgs := append([]interface{}{}, keysArgs...)
	if keysWhere != "" {
		whereParts = append(whereParts, keysWhere)
	}
	if filter != nil && filter.Origin != nil {
		whereParts = append(whereParts, "t.origin = ?")
		whereArgs = append(whereArgs, string(*filter.Origin))
	}

	where := ""
	if len(whereParts) > 0 {
		where = " WHERE " + strings.Join(whereParts, " AND ")
	}
	from := " FROM " + tombstoneTable + " t" + where

	// the counts per origin ignore the origin filter
	var counts struct {
		Manual int `db:"manual"`
		Clean  int `db:"clean"`
	}
	countQuery := "SELECT COALESCE(SUM(t.origin = ?), 0) AS manual, COALESCE(SUM(t.origin = ?), 0) AS clean FROM " + tombstoneTable + " t"
	countArgs := []interface{}{string(models.TombstoneOriginManual), string(models.TombstoneOriginClean)}
	if keysWhere != "" {
		countQuery += " WHERE " + keysWhere
		countArgs = append(countArgs, keysArgs...)
	}
	if err := dbWrapper.Get(ctx, &counts, countQuery, countArgs...); err != nil {
		return nil, fmt.Errorf("counting tombstones: %w", err)
	}

	var count int
	if err := dbWrapper.Get(ctx, &count, "SELECT COUNT(*)"+from, whereArgs...); err != nil {
		return nil, fmt.Errorf("counting tombstones: %w", err)
	}

	// select the id plus one 0/1 column per key to report matched keys
	cols := []string{"t.id"}
	var selectArgs []interface{}
	for _, c := range conds {
		cols = append(cols, "COALESCE(("+c.sql+"), 0)")
		selectArgs = append(selectArgs, c.args...)
	}

	query := "SELECT " + strings.Join(cols, ", ") + from + orderBy
	args := make([]interface{}, 0, len(selectArgs)+len(whereArgs)+2)
	args = append(args, selectArgs...)
	args = append(args, whereArgs...)
	if !findFilter.IsGetAll() {
		query += " LIMIT ? OFFSET ?"
		pageSize := findFilter.GetPageSize()
		args = append(args, pageSize, (findFilter.GetPage()-1)*pageSize)
	}

	ids, matchedKeys, err := queryTombstoneMatches(ctx, query, args, conds)
	if err != nil {
		return nil, err
	}

	tombstones, err := qb.findMany(ctx, ids)
	if err != nil {
		return nil, err
	}

	ret := &models.TombstoneQueryResult{
		Count:       count,
		ManualCount: counts.Manual,
		CleanCount:  counts.Clean,
		Matches:     make([]*models.TombstoneMatch, len(tombstones)),
	}
	for i, t := range tombstones {
		ret.Matches[i] = &models.TombstoneMatch{Tombstone: t, MatchedKeys: matchedKeys[t.ID]}
	}

	return ret, nil
}

// queryTombstoneMatches runs the match query and returns the tombstone ids in
// result order plus the keys each one matched on. The rows are closed before
// it returns, so the caller can run further queries.
func queryTombstoneMatches(ctx context.Context, query string, args []interface{}, conds []tombstoneCondition) ([]int, map[int][]models.TombstoneMatchKey, error) {
	rows, err := dbWrapper.Queryx(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("querying tombstones: %w", err)
	}
	defer rows.Close()

	var ids []int
	matchedKeys := make(map[int][]models.TombstoneMatchKey)
	for rows.Next() {
		var id int
		flags := make([]bool, len(conds))
		dest := []interface{}{&id}
		for i := range flags {
			dest = append(dest, &flags[i])
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, nil, fmt.Errorf("scanning tombstone match: %w", err)
		}
		ids = append(ids, id)
		for i, matched := range flags {
			if matched {
				matchedKeys[id] = append(matchedKeys[id], conds[i].key)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("querying tombstones: %w", err)
	}

	return ids, matchedKeys, nil
}
