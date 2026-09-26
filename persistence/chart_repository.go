package persistence

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

type chartRepository struct {
	sqlRepository
	mediaFileRepo model.MediaFileRepository
}

func NewChartRepository(db dbx.Builder) model.ChartRepository {
	r := &chartRepository{}
	r.db = db
	r.tableName = "scrobbles"
	r.mediaFileRepo = NewMediaFileRepository(db)

	return r
}

func (r *chartRepository) GetChart(ctx context.Context, options model.ChartOptions) ([]model.ChartEntry, error) {
	curStart, curEnd, prevStart, prevEnd := calendarWindow(options)

	curRows, err := r.scrobblesInWindow(
		ctx,
		options.UserID,
		options.Scope,
		curStart,
		curEnd,
	)
	if err != nil {
		return nil, err
	}

	if len(curRows) == 0 {
		return []model.ChartEntry{}, nil
	}

	prevRows, err := r.scrobblesInWindow(
		ctx,
		options.UserID,
		options.Scope,
		prevStart,
		prevEnd,
	)
	if err != nil {
		return nil, err
	}

	prevRank := rowsToRank(prevRows)
	curChart := rowsToChart(curRows, prevRank)

	limitIdx := len(curChart)
	for idx, entry := range curChart {
		if entry.Rank > options.Limit {
			limitIdx = idx
			break
		}
	}
	curChart = curChart[:limitIdx]

	topIDs := make([]string, 0, len(curChart))
	for _, entry := range curChart {
		topIDs = append(topIDs, entry.Song.ID)
	}

	mediaFiles, err := r.mediaFileRepo.GetAll(ctx, model.QueryOptions{
		Filters: sq.Eq{"media_file.id": topIDs},
	})
	if err != nil {
		return nil, err
	}

	mfByID := make(map[string]model.MediaFile, len(mediaFiles))
	for _, mf := range mediaFiles {
		mfByID[mf.ID] = mf
	}

	for idx := range len(curChart) {
		mf, ok := mfByID[curChart[idx].Song.ID]
		if !ok {
			continue
		}
		curChart[idx].Song = mf
	}

	return curChart, nil
}

func (r *chartRepository) scrobblesInWindow(
	ctx context.Context,
	userID string,
	scope model.ChartScope,
	start time.Time,
	end time.Time,
) ([]scrobbleCountRow, error) {
	sel := sq.Select(
		"media_file_id",
		"count(*) as play_count",
		"min(submission_time) as first_played",
	).
		From("scrobbles").
		Where(sq.GtOrEq{"submission_time": start.Unix()}).
		Where(sq.Lt{"submission_time": end.Unix()}).
		GroupBy("media_file_id").
		OrderBy("play_count DESC", "first_played ASC", "media_file_id ASC")

	if scope == model.ChartScopePersonal {
		sel = sel.Where(sq.Eq{"user_id": userID})
	}

	var rows []scrobbleCountRow
	err := r.queryAll(ctx, sel, &rows)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

// calendarWindow returns (curStart, curEnd, prevStart, prevEnd) for the
// requested calendar period, inferred from whether Month or Week is set.
// Ranges are half-open: [start, end).
func calendarWindow(opts model.ChartOptions) (time.Time, time.Time, time.Time, time.Time) {
	if opts.Month == 0 && opts.Week == 0 {
		curStart := time.Date(opts.Year, time.January, 1, 0, 0, 0, 0, time.UTC)
		curEnd := curStart.AddDate(1, 0, 0)
		prevStart := curStart.AddDate(-1, 0, 0)
		prevEnd := curStart
		return curStart, curEnd, prevStart, prevEnd
	}
	if opts.Month != 0 {
		curStart := time.Date(opts.Year, time.Month(opts.Month), 1, 0, 0, 0, 0, time.UTC)
		curEnd := curStart.AddDate(0, 1, 0)
		prevStart := curStart.AddDate(0, -1, 0)
		prevEnd := curStart
		return curStart, curEnd, prevStart, prevEnd
	}

	// Week period.
	curStart := isoWeekStart(opts.Year, opts.Week)
	curEnd := curStart.AddDate(0, 0, 7)
	prevStart := curStart.AddDate(0, 0, -7)
	prevEnd := curStart

	return curStart, curEnd, prevStart, prevEnd
}

// isoWeekStart returns the Monday 00:00:00 UTC that begins the given
// ISO-8601 week/year (week 1 is the week containing the year's first Thursday).
func isoWeekStart(year, week int) time.Time {
	jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, time.UTC)
	offset := int(jan4.Weekday())
	if offset == 0 { // Sunday
		offset = 7
	}
	week1Monday := jan4.AddDate(0, 0, -(offset - 1))

	return week1Monday.AddDate(0, 0, (week-1)*7)
}

type scrobbleCountRow struct {
	MediaFileID  string `db:"media_file_id"`
	PlayCount    int    `db:"play_count"`
	FirstPlayed  int64  `db:"first_played"`
}

func rowsToChart(rows []scrobbleCountRow, prevRanks map[string]int) (chart []model.ChartEntry) {
	curRank := 1
	curPlayCount := -1
	chart = make([]model.ChartEntry, 0, len(rows))
	for idx, row := range rows {
		if row.PlayCount != curPlayCount {
			curRank = idx + 1
			curPlayCount = row.PlayCount
		}
		entry := model.ChartEntry{
			PlayCount: row.PlayCount,
			Rank:      curRank,
			Song:      model.MediaFile{ID: row.MediaFileID},
		}
		if pr, ok := prevRanks[row.MediaFileID]; ok {
			entry.PreviousRank = new(pr)
		}

		chart = append(chart, entry)
	}

	return chart
}

func rowsToRank(rows []scrobbleCountRow) map[string]int {
	rankByID := make(map[string]int, len(rows))
	curRank := 1
	curPlayCount := -1
	for idx, row := range rows {
		if row.PlayCount != curPlayCount {
			curRank = idx + 1
			curPlayCount = row.PlayCount
		}
		rankByID[row.MediaFileID] = curRank
	}

	return rankByID
}
