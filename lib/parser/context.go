package parser

import (
	"github.com/1119-Labs/callisto/v4/lib/logging"
	"github.com/1119-Labs/callisto/v4/lib/node"

	"github.com/1119-Labs/callisto/v4/lib/database"
	"github.com/1119-Labs/callisto/v4/lib/modules"
)

// Context represents the context that is shared among different workers
type Context struct {
	Node     node.Node
	Database database.Database
	Logger   logging.Logger
	Modules  []modules.Module
}

// NewContext builds a new Context instance
func NewContext(
	proxy node.Node, db database.Database,
	logger logging.Logger, modules []modules.Module,
) *Context {
	return &Context{
		Node:     proxy,
		Database: db,
		Modules:  modules,
		Logger:   logger,
	}
}
