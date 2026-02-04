package v5

import (
	"fmt"

	parse "github.com/1119-Labs/callisto/v4/lib/cmd/parse/types"
	"github.com/1119-Labs/callisto/v4/lib/database"
	"github.com/1119-Labs/callisto/v4/lib/database/postgresql"
	"github.com/1119-Labs/callisto/v4/lib/types/config"

	v5db "github.com/1119-Labs/callisto/v4/database/migrate/v5"
)

// RunMigration runs the migrations to v5
func RunMigration(parseConfig *parse.Config) error {
	cfg, err := GetConfig()
	if err != nil {
		return fmt.Errorf("error while reading config: %s", err)
	}

	// Migrate the database
	err = migrateDb(cfg, parseConfig)
	if err != nil {
		return fmt.Errorf("error while migrating database: %s", err)
	}

	return nil
}

func migrateDb(cfg config.Config, parseConfig *parse.Config) error {
	// Get the db
	databaseCtx := database.NewContext(cfg.Database, parseConfig.GetLogger())
	db, err := postgresql.Builder(databaseCtx)
	if err != nil {
		return fmt.Errorf("error while building the db: %s", err)
	}

	// Build the migrator and perform the migrations
	migrator := v5db.NewMigrator(db.(*postgresql.Database))
	return migrator.Migrate()
}
