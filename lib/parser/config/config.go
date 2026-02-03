package config

import "time"

type Config struct {
	GenesisFilePath string         `yaml:"genesis_file_path,omitempty"`
	BlockWorkers    int64          `yaml:"block_workers"`
	TxWorkers       int64          `yaml:"tx_workers"`
	StartHeight     int64          `yaml:"start_height"`
	AvgBlockTime    *time.Duration `yaml:"average_block_time"`
	ParseNewBlocks  bool           `yaml:"listen_new_blocks"`
	ParseOldBlocks  bool           `yaml:"parse_old_blocks"`
	ParseGenesis    bool           `yaml:"parse_genesis"`
	FastSync        bool           `yaml:"fast_sync,omitempty"`
}

// NewParsingConfig allows to build a new Config instance
func NewParsingConfig(
	blockWorkers int64,
	txWorkers int64,
	parseNewBlocks, parseOldBlocks bool,
	parseGenesis bool, genesisFilePath string,
	startHeight int64, fastSync bool,
	avgBlockTime *time.Duration,
) Config {
	return Config{
		BlockWorkers:    blockWorkers,
		TxWorkers:       txWorkers,
		ParseOldBlocks:  parseOldBlocks,
		ParseNewBlocks:  parseNewBlocks,
		ParseGenesis:    parseGenesis,
		GenesisFilePath: genesisFilePath,
		StartHeight:     startHeight,
		FastSync:        fastSync,
		AvgBlockTime:    avgBlockTime,
	}
}

// DefaultParsingConfig returns the default instance of Config
func DefaultParsingConfig() Config {
	avgBlockTime := 5 * time.Second
	return NewParsingConfig(
		4, // block_workers
		8, // tx_workers (typically more since tx processing is heavier)
		true,
		true,
		true,
		"",
		1,
		false,
		&avgBlockTime,
	)
}
