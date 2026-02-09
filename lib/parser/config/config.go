package config

import "time"

type Config struct {
	GenesisFilePath string         `yaml:"genesis_file_path,omitempty"`
	StartHeight     int64          `yaml:"start_height"`
	AvgBlockTime    *time.Duration `yaml:"average_block_time"`
	ParseNewBlocks  bool           `yaml:"listen_new_blocks"`
	ParseOldBlocks  bool           `yaml:"parse_old_blocks"`
	ParseGenesis    bool           `yaml:"parse_genesis"`
	FastSync        bool           `yaml:"fast_sync,omitempty"`

	// New block pipeline workers (high priority - real-time)
	NewBlockWorkers int64 `yaml:"new_block_workers"`

	// Old/missing block pipeline workers (lower priority - backfill)
	OldBlockWorkers int64 `yaml:"old_block_workers"`
}

// NewParsingConfig allows to build a new Config instance
func NewParsingConfig(
	newBlockWorkers int64,
	oldBlockWorkers int64,
	parseNewBlocks, parseOldBlocks bool,
	parseGenesis bool, genesisFilePath string,
	startHeight int64, fastSync bool,
	avgBlockTime *time.Duration,
) Config {
	return Config{
		NewBlockWorkers: newBlockWorkers,
		OldBlockWorkers: oldBlockWorkers,
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
		2, // new_block_workers (priority, fewer needed for real-time)
		4, // old_block_workers (more workers for backfill)
		true,
		true,
		true,
		"",
		1,
		false,
		&avgBlockTime,
	)
}
