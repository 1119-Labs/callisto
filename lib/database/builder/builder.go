package builder

import (
	"github.com/1119-Labs/callisto/v4/lib/database"

	"github.com/1119-Labs/callisto/v4/lib/database/postgresql"
)

// Builder represents a generic Builder implementation that build the proper database
// instance based on the configuration the user has specified
func Builder(ctx *database.Context) (database.Database, error) {
	return postgresql.Builder(ctx)
}
