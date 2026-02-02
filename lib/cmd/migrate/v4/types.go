package v4

import (
	databaseconfig "github.com/1119-Labs/callisto/v4/lib/database/config"
	loggingconfig "github.com/1119-Labs/callisto/v4/lib/logging/config"
	"github.com/1119-Labs/callisto/v4/lib/modules/pruning"
	"github.com/1119-Labs/callisto/v4/lib/modules/telemetry"
	nodeconfig "github.com/1119-Labs/callisto/v4/lib/node/config"
	parserconfig "github.com/1119-Labs/callisto/v4/lib/parser/config"
	pricefeedconfig "github.com/1119-Labs/callisto/v4/lib/pricefeed"
	"github.com/1119-Labs/callisto/v4/lib/types/config"
)

// Config defines all necessary juno configuration parameters.
type Config struct {
	Chain    config.ChainConfig    `yaml:"chain"`
	Node     nodeconfig.Config     `yaml:"node"`
	Parser   parserconfig.Config   `yaml:"parsing"`
	Database databaseconfig.Config `yaml:"database"`
	Logging  loggingconfig.Config  `yaml:"logging"`

	// The following are there to support modules which config are present if they are enabled

	Telemetry *telemetry.Config       `yaml:"telemetry,omitempty"`
	Pruning   *pruning.Config         `yaml:"pruning,omitempty"`
	PriceFeed *pricefeedconfig.Config `yaml:"pricefeed,omitempty"`
}
