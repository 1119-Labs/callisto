package daily_refetch

import (
	"github.com/1119-Labs/callisto/v4/lib/node"

	callistodb "github.com/1119-Labs/callisto/v4/database"

	"github.com/1119-Labs/callisto/v4/lib/modules"
)

var (
	_ modules.Module                   = &Module{}
	_ modules.PeriodicOperationsModule = &Module{}
)

type Module struct {
	node     node.Node
	database *callistodb.Db
}

// NewModule builds a new Module instance
func NewModule(
	node node.Node,
	database *callistodb.Db,
) *Module {
	return &Module{
		node:     node,
		database: database,
	}
}

// Name implements modules.Module
func (m *Module) Name() string {
	return "daily refetch"
}
