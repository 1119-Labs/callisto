package config

import (
	"strings"

	"gopkg.in/yaml.v3"

	databaseconfig "github.com/1119-Labs/callisto/v4/lib/database/config"
	loggingconfig "github.com/1119-Labs/callisto/v4/lib/logging/config"
	nodeconfig "github.com/1119-Labs/callisto/v4/lib/node/config"
	parserconfig "github.com/1119-Labs/callisto/v4/lib/parser/config"
)

var (
	// Cfg represents the configuration to be used during the execution
	Cfg Config
)

// Config defines all necessary juno configuration parameters.
type Config struct {
	bytes []byte

	Chain    ChainConfig           `yaml:"chain"`
	Node     nodeconfig.Config     `yaml:"node"`
	Parser   parserconfig.Config   `yaml:"parsing"`
	Queue    QueueConfig           `yaml:"queue"`
	Database databaseconfig.Config `yaml:"database"`
	Logging  loggingconfig.Config  `yaml:"logging"`
}

// NewConfig builds a new Config instance
func NewConfig(
	nodeCfg nodeconfig.Config,
	chainCfg ChainConfig, dbConfig databaseconfig.Config,
	parserConfig parserconfig.Config, queueConfig QueueConfig, loggingConfig loggingconfig.Config,
) Config {
	return Config{
		Node:     nodeCfg,
		Chain:    chainCfg,
		Database: dbConfig,
		Parser:   parserConfig,
		Queue:    queueConfig,
		Logging:  loggingConfig,
	}
}

func DefaultConfig() Config {
	cfg := NewConfig(
		nodeconfig.DefaultConfig(),
		DefaultChainConfig(), databaseconfig.DefaultDatabaseConfig(),
		parserconfig.DefaultParsingConfig(), DefaultQueueConfig(), loggingconfig.DefaultLoggingConfig(),
	)

	bz, err := yaml.Marshal(cfg)
	if err != nil {
		panic(err)
	}

	cfg.bytes = bz
	return cfg
}

func (c Config) GetBytes() ([]byte, error) {
	return c.bytes, nil
}

// ---------------------------------------------------------------------------------------------------------------------

type ChainConfig struct {
	Bech32Prefix string   `yaml:"bech32_prefix"`
	Modules      []string `yaml:"modules"`
}

// NewChainConfig returns a new ChainConfig instance
func NewChainConfig(bech32Prefix string, modules []string) ChainConfig {
	return ChainConfig{
		Bech32Prefix: bech32Prefix,
		Modules:      modules,
	}
}

// DefaultChainConfig returns the default instance of ChainConfig
func DefaultChainConfig() ChainConfig {
	return NewChainConfig("cosmos", nil)
}

func (cfg ChainConfig) IsModuleEnabled(moduleName string) bool {
	for _, module := range cfg.Modules {
		if strings.EqualFold(module, moduleName) {
			return true
		}
	}

	return false
}

// ---------------------------------------------------------------------------------------------------------------------

// QueueConfig contains queue-related configuration.
type QueueConfig struct {
	RabbitMQ RabbitMQConfig `yaml:"rabbitmq"`
}

// RabbitMQConfig contains RabbitMQ connection and queue settings.
type RabbitMQConfig struct {
	URL            string `yaml:"url"`
	BlockQueueName string `yaml:"block_queue_name"`
	TxQueueName    string `yaml:"tx_queue_name"`
	Prefetch       int    `yaml:"prefetch"`
}

// DefaultQueueConfig returns the default queue configuration.
func DefaultQueueConfig() QueueConfig {
	return QueueConfig{
		RabbitMQ: RabbitMQConfig{
			URL:            "amqp://guest:guest@localhost:5672/",
			BlockQueueName: "callisto-block-queue",
			TxQueueName:    "callisto-tx-queue",
			Prefetch:       25,
		},
	}
}
