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
	NewTxWorkers    int64 `yaml:"new_tx_workers"`

	// Old/missing block pipeline workers (lower priority - backfill)
	OldBlockWorkers int64 `yaml:"old_block_workers"`
	OldTxWorkers    int64 `yaml:"old_tx_workers"`
}

// NewParsingConfig allows to build a new Config instance
func NewParsingConfig(
	newBlockWorkers int64,
	newTxWorkers int64,
	oldBlockWorkers int64,
	oldTxWorkers int64,
	parseNewBlocks, parseOldBlocks bool,
	parseGenesis bool, genesisFilePath string,
	startHeight int64, fastSync bool,
	avgBlockTime *time.Duration,
) Config {
	return Config{
		NewBlockWorkers: newBlockWorkers,
		NewTxWorkers:    newTxWorkers,
		OldBlockWorkers: oldBlockWorkers,
		OldTxWorkers:    oldTxWorkers,
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
		2,  // new_block_workers (priority, fewer needed for real-time)
		10, // new_tx_workers (handle new block transactions quickly)
		4,  // old_block_workers (more workers for backfill)
		20, // old_tx_workers (heavy lifting for historical data)
		true,
		true,
		true,
		"",
		1,
		false,
		&avgBlockTime,
	)
}
