package main

import (
	"fmt"

	"gorm.io/gorm"
)

const migration008SchemaSQL = `
ALTER TABLE streamers ADD COLUMN IF NOT EXISTS agency_user_id UUID;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_streamers_agency_user_id'
    ) THEN
        ALTER TABLE streamers
        ADD CONSTRAINT fk_streamers_agency_user_id
        FOREIGN KEY (agency_user_id) REFERENCES users(id);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_streamers_agency_user_id ON streamers (agency_user_id);
`

func applyStreamerAgencyMigration(db *gorm.DB) error {
	if err := db.Exec(migration008SchemaSQL).Error; err != nil {
		return fmt.Errorf("execute migration 008 schema: %w", err)
	}
	if err := failOnAgencyBackfillConflicts(db); err != nil {
		return err
	}
	if err := backfillStreamerAgencyUserID(db); err != nil {
		return fmt.Errorf("backfill streamer agency_user_id: %w", err)
	}
	return nil
}

func failOnAgencyBackfillConflicts(db *gorm.DB) error {
	type conflictRow struct {
		ChannelID string
	}

	var conflicts []conflictRow
	if err := db.Table("agency_streamers").
		Select("channel_id").
		Group("channel_id").
		Having("COUNT(DISTINCT agency_id) > 1").
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
		sql = `
			UPDATE streamers s
			SET agency_user_id = src.agency_id
			FROM (
				SELECT channel_id, agency_id
				FROM agency_streamers
				GROUP BY channel_id, agency_id
			) AS src
			WHERE s.channel_id = src.channel_id
			  AND s.agency_user_id IS NULL
		`
	}
	if err := db.Exec(sql).Error; err != nil {
		return fmt.Errorf("update streamer agency_user_id from legacy mappings: %w", err)
	}
	return nil
}
