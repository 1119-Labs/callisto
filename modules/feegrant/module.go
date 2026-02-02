package feegrant

import (
	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/1119-Labs/callisto/v4/database"

	"github.com/1119-Labs/callisto/v4/lib/modules"
)

var (
	_ modules.BlockModule        = &Module{}
	_ modules.Module             = &Module{}
	_ modules.MessageModule      = &Module{}
	_ modules.AuthzMessageModule = &Module{}
)

// Module represent x/feegrant module
type Module struct {
	cdc codec.Codec
	db  *database.Db
}

// NewModule returns a new Module instance
func NewModule(cdc codec.Codec, db *database.Db) *Module {
	return &Module{
		cdc: cdc,
		db:  db,
	}
}

// Name implements modules.Module
func (m *Module) Name() string {
	return "feegrant"
}
