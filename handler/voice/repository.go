package voice

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type UpdateHitCacheAndReturnAudio func(ctx context.Context, logger *zap.Logger, cacheKey string) (string, error)

func NewUpdateHitCacheAndReturnAudio(db *pgxpool.Pool) UpdateHitCacheAndReturnAudio {
	return func(ctx context.Context, logger *zap.Logger, cacheKey string) (string, error) {
		var audioUrl string
		const updateCacheAndGetAudioUrl = `
			update tbl_tts_cache
			set hit_count=hit_count+1 ,
				last_accessed_at = now()
			where cache_key = $1
			returning audio_url
		`

		rows, err := db.Query(ctx, updateCacheAndGetAudioUrl, cacheKey)
		if err != nil {
			logger.Error(err.Error())
			return audioUrl, errors.New("failed to check daily plans")
		}

		for rows.Next() {
			err := rows.Scan(&audioUrl)
			if err != nil {
				return audioUrl, err
			}
		}

		return audioUrl, nil
	}
}

type InsertAudioUrlAndKeyToCacheFunc func(ctx context.Context, logger *zap.Logger, cacheKey, text, audioUrl, key string) error

func NewInsertAudioUrlAndKeyToCacheFunc(db *pgxpool.Pool) InsertAudioUrlAndKeyToCacheFunc {
	return func(ctx context.Context, logger *zap.Logger, cacheKey, text, audioUrl, key string) error {
		sql := `
			insert into tbl_tts_cache (cache_key, text, voice, speed,audio_url, last_accessed_at, audio_key)
			values ($1,$2,'DEFAULT',1.0,$3,now(),$4);
		`
		_, err := db.Exec(ctx, sql, cacheKey, text, audioUrl, key)
		if err != nil {
			return err
		}

		return nil
	}
}
