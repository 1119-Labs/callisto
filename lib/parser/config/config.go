package config

import "time"

type Config struct {
	GenesisFilePath string         `yaml:"genesis_file_path,omitempty"`
	StartHeight     int64          `yaml:"start_height"`
	AvgBlockTime    *time.Duration `yaml:"average_block_time"`
	ParseNewBlocks  bool           `yaml:"listen_new_blocks"`
	ParseOldBlocks  bool           `yaml:"parse_old_blocks"`
	ParseGenesis    bool           `yaml:"parse_genesis"`

	// New block pipeline workers (high priority - real-time)
	NewBlockWorkers int64 `yaml:"new_block_workers"`

	// Old/missing block pipeline workers (lower priority - backfill)
	OldBlockWorkers int64 `yaml:"old_block_workers"`

	// Dead-letter queue retry workers (optional, disabled by default)
	ParseDLQ   bool           `yaml:"parse_dead_letter_queue"`
	DLQWorkers int64          `yaml:"dlq_workers"`
	DLQMinAge  *time.Duration `yaml:"dlq_min_age,omitempty"` // minimum time a message must sit in DLQ before retry (default 5m)

	// Delay between enqueueing missing blocks
	MissingBlockEnqueueDelay *time.Duration `yaml:"missing_block_enqueue_delay,omitempty"`
}

// NewParsingConfig allows to build a new Config instance
func NewParsingConfig(
	newBlockWorkers int64,
	oldBlockWorkers int64,
	parseNewBlocks, parseOldBlocks bool,
	parseGenesis bool, genesisFilePath string,
	startHeight int64,
	avgBlockTime *time.Duration,
	missingBlockEnqueueDelay *time.Duration,
) Config {
	dlqMinAge := 5 * time.Minute
	return Config{
		NewBlockWorkers:          newBlockWorkers,
		OldBlockWorkers:          oldBlockWorkers,
		ParseDLQ:                 false,
		DLQWorkers:               1,
		DLQMinAge:                &dlqMinAge,
		ParseOldBlocks:           parseOldBlocks,
		ParseNewBlocks:           parseNewBlocks,
		ParseGenesis:             parseGenesis,
		GenesisFilePath:          genesisFilePath,
		StartHeight:              startHeight,
		AvgBlockTime:             avgBlockTime,
		MissingBlockEnqueueDelay: missingBlockEnqueueDelay,
	}
}

// DefaultParsingConfig returns the default instance of Config
func DefaultParsingConfig() Config {
	avgBlockTime := 5 * time.Second
	missingBlockDelay := 5 * time.Second
	return NewParsingConfig(
		2, // new_block_workers (priority, fewer needed for real-time)
		4, // old_block_workers (more workers for backfill)
		true,
		true,
		true,
		"",
		1,
		&avgBlockTime,
		&missingBlockDelay,
	)
}
