package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/donder-core/hiroba-scraper-service/internal/parser/models"
)


const upsertScoreDetailQuery = `
INSERT INTO score_detail (
    song_no, level, taiko_no,
    crown_src, best_score_icon_src, ranking,
    high_score, good, combo, ok, drumroll, bad
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
ON CONFLICT (song_no, level, taiko_no) DO UPDATE SET
    crown_src           = EXCLUDED.crown_src,
    best_score_icon_src = EXCLUDED.best_score_icon_src,
    ranking             = EXCLUDED.ranking,
    high_score          = EXCLUDED.high_score,
    good                = EXCLUDED.good,
    combo               = EXCLUDED.combo,
    ok                  = EXCLUDED.ok,
    drumroll            = EXCLUDED.drumroll,
    bad                 = EXCLUDED.bad
RETURNING crown_src, best_score_icon_src, ranking, high_score, good, combo, ok, drumroll, bad`

type PostgresScoreRepository struct {
	db *sql.DB
}

func NewPostgresScoreRepository(db *sql.DB) *PostgresScoreRepository {
	return &PostgresScoreRepository{db: db}
}

func (r *PostgresScoreRepository) UpsertScoreDetail(ctx context.Context, songNo, level int, taikoNo string, detail *models.ScoreDetail) (*models.ScoreDetail, error) {
	row := r.db.QueryRowContext(ctx, upsertScoreDetailQuery,
		songNo, level, taikoNo,
		detail.CrownSrc, detail.BestScoreIconSrc, detail.Ranking,
		detail.HighScore, detail.Good, detail.Combo, detail.OK, detail.Drumroll, detail.Bad,
	)

	var result models.ScoreDetail
	if err := row.Scan(
		&result.CrownSrc, &result.BestScoreIconSrc, &result.Ranking,
		&result.HighScore, &result.Good, &result.Combo, &result.OK, &result.Drumroll, &result.Bad,
	); err != nil {
		return nil, fmt.Errorf("upsert score_detail (song_no=%d, level=%d, taiko_no=%s): %w", songNo, level, taikoNo, err)
	}

	return &result, nil
}

func (r *PostgresScoreRepository) BatchUpsertScoreDetails(ctx context.Context, taikoNo string, records []models.ScoredRecord) error {
	if len(records) == 0 {
		return nil
	}

	// Build a multi-row VALUES clause: ($1,$2,...,$12), ($13,$14,...,$24), ...
	const colsPerRow = 12
	placeholders := make([]string, len(records))
	args := make([]any, 0, len(records)*colsPerRow)

	for i, r := range records {
		base := i * colsPerRow
		placeholders[i] = fmt.Sprintf(
			"($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
			base+1, base+2, base+3, base+4, base+5, base+6,
			base+7, base+8, base+9, base+10, base+11, base+12,
		)
		args = append(args,
			r.SongNo, r.Level, taikoNo,
			r.Detail.CrownSrc, r.Detail.BestScoreIconSrc, r.Detail.Ranking,
			r.Detail.HighScore, r.Detail.Good, r.Detail.Combo, r.Detail.OK, r.Detail.Drumroll, r.Detail.Bad,
		)
	}

	query := fmt.Sprintf(`
INSERT INTO score_detail (
    song_no, level, taiko_no,
    crown_src, best_score_icon_src, ranking,
    high_score, good, combo, ok, drumroll, bad
) VALUES %s
ON CONFLICT (song_no, level, taiko_no) DO UPDATE SET
    crown_src           = EXCLUDED.crown_src,
    best_score_icon_src = EXCLUDED.best_score_icon_src,
    ranking             = EXCLUDED.ranking,
    high_score          = EXCLUDED.high_score,
    good                = EXCLUDED.good,
    combo               = EXCLUDED.combo,
    ok                  = EXCLUDED.ok,
    drumroll            = EXCLUDED.drumroll,
    bad                 = EXCLUDED.bad`,
		strings.Join(placeholders, ","),
	)

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("batch upsert score_detail (%d records, taiko_no=%s): %w", len(records), taikoNo, err)
	}
	return nil
}