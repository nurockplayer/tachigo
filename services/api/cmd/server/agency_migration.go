package main

import (
	"fmt"

	"gorm.io/gorm"
)

func failOnAgencyBackfillConflicts(db *gorm.DB) error {
	type conflictRow struct {
		ChannelID string
	}

	var conflicts []conflictRow
	if err := db.Table("agency_streamers").
		Select("agency_streamers.channel_id").
		Joins("JOIN streamers ON streamers.channel_id = agency_streamers.channel_id").
		Where("streamers.agency_user_id IS NULL").
		Group("agency_streamers.channel_id").
		Having("COUNT(DISTINCT agency_streamers.agency_id) > 1").
		Find(&conflicts).Error; err != nil {
		return fmt.Errorf("detect agency backfill conflicts: %w", err)
	}
	if len(conflicts) == 0 {
		return nil
	}
	return fmt.Errorf(
		"agency backfill conflict: %d channel(s) map to multiple agencies in agency_streamers; resolve before deploying",
		len(conflicts),
	)
}

const postgresBackfillStreamerAgencyUserIDSQL = `
	UPDATE streamers s
	SET agency_user_id = src.agency_id
	FROM (
		SELECT DISTINCT ON (channel_id) channel_id, agency_id
		FROM agency_streamers
		ORDER BY channel_id
	) AS src
	WHERE s.channel_id = src.channel_id
	  AND s.agency_user_id IS NULL
`

func backfillStreamerAgencyUserID(db *gorm.DB) error {
	sql := `
		UPDATE streamers
		SET agency_user_id = (
			SELECT agency_streamers.agency_id
			FROM agency_streamers
			WHERE agency_streamers.channel_id = streamers.channel_id
			LIMIT 1
		)
		WHERE agency_user_id IS NULL
		  AND EXISTS (
			SELECT 1
			FROM agency_streamers
			WHERE agency_streamers.channel_id = streamers.channel_id
		)
	`
	if db.Dialector.Name() == "postgres" {
		sql = postgresBackfillStreamerAgencyUserIDSQL
	}
	if err := db.Exec(sql).Error; err != nil {
		return fmt.Errorf("update streamer agency_user_id from legacy mappings: %w", err)
	}
	return nil
}
