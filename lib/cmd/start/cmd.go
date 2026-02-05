package start

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	parsecmdtypes "github.com/1119-Labs/callisto/v4/lib/cmd/parse/types"
	"github.com/1119-Labs/callisto/v4/lib/modules"
	"github.com/1119-Labs/callisto/v4/lib/types/utils"

	"github.com/1119-Labs/callisto/v4/lib/logging"

	"github.com/1119-Labs/callisto/v4/lib/queue"
	"github.com/1119-Labs/callisto/v4/lib/types"
	"github.com/1119-Labs/callisto/v4/lib/types/config"

	"github.com/go-co-op/gocron"

	"github.com/1119-Labs/callisto/v4/lib/parser"

	"github.com/spf13/cobra"
)

var (
	waitGroup sync.WaitGroup
)

// Closeable interface for resources that can be closed
type Closeable interface {
	Close() error
}

// NewStartCmd returns the command that should be run when we want to start parsing a chain state.
func NewStartCmd(cmdCfg *parsecmdtypes.Config) *cobra.Command {
	return &cobra.Command{
		Use:     "start",
		Short:   "Start parsing the blockchain data",
		PreRunE: parsecmdtypes.ReadConfigPreRunE(cmdCfg),
		RunE: func(cmd *cobra.Command, args []string) error {
			context, err := parsecmdtypes.GetParserContext(config.Cfg, cmdCfg)
			if err != nil {
				return err
			}

			// Run all the additional operations
			for _, module := range context.Modules {
				if module, ok := module.(modules.AdditionalOperationsModule); ok {
					err = module.RunAdditionalOperations()
					if err != nil {
						return err
					}
				}
			}

			return startParsing(context)
		},
	}
}

// startParsing represents the function that should be called when the parse command is executed
func startParsing(ctx *parser.Context) error {
	// Get the config
	cfg := config.Cfg.Parser
	logging.StartHeight.Add(float64(cfg.StartHeight))

	// Start periodic operations
	scheduler := gocron.NewScheduler(time.UTC)
	for _, module := range ctx.Modules {
		if module, ok := module.(modules.PeriodicOperationsModule); ok {
			err := module.RegisterPeriodicOperations(scheduler)
			if err != nil {
				return err
			}
		}
	}
	scheduler.StartAsync()

	queueCfg := config.Cfg.Queue.RabbitMQ

	// Track all connections for graceful shutdown
	var allConnections []io.Closer

	// =============================================================
	// NEW BLOCK PIPELINE (high priority - real-time blocks)
	// =============================================================

	// Publisher for enqueueing new block heights
	newBlockPublisher, err := queue.ConnectNewBlockQueue(queueCfg)
	if err != nil {
		return err
	}
	allConnections = append(allConnections, newBlockPublisher)

	// New block workers: consume heights, save block data, publish tx hashes to new-tx-queue
	newBlockWorkers := make([]parser.BlockWorker, cfg.NewBlockWorkers)
	for i := range newBlockWorkers {
		blockConsumer, err := queue.ConnectNewBlockQueue(queueCfg)
		if err != nil {
			return err
		}
		allConnections = append(allConnections, blockConsumer)

		// Each block worker gets its own tx publisher connection
		txPublisher, err := queue.ConnectNewTxQueue(queueCfg)
		if err != nil {
			return err
		}
		allConnections = append(allConnections, txPublisher)

		newBlockWorkers[i] = parser.NewBlockWorker(ctx, blockConsumer, txPublisher, int(i), "new")
	}

	// New tx workers: consume from new-tx-queue
	newTxWorkers := make([]parser.TxWorker, cfg.NewTxWorkers)
	for i := range newTxWorkers {
		txConsumer, err := queue.ConnectNewTxQueue(queueCfg)
		if err != nil {
			return err
		}
		allConnections = append(allConnections, txConsumer)
		newTxWorkers[i] = parser.NewTxWorker(ctx, txConsumer, int(i), "new")
	}

	// =============================================================
	// OLD BLOCK PIPELINE (lower priority - backfill/missing blocks)
	// =============================================================

	// Publisher for enqueueing old/missing block heights
	oldBlockPublisher, err := queue.ConnectOldBlockQueue(queueCfg)
	if err != nil {
		return err
	}
	allConnections = append(allConnections, oldBlockPublisher)

	// Old block workers: consume heights, save block data, publish tx hashes to old-tx-queue
	oldBlockWorkers := make([]parser.BlockWorker, cfg.OldBlockWorkers)
	for i := range oldBlockWorkers {
		blockConsumer, err := queue.ConnectOldBlockQueue(queueCfg)
		if err != nil {
			return err
		}
		allConnections = append(allConnections, blockConsumer)

		// Each block worker gets its own tx publisher connection
		txPublisher, err := queue.ConnectOldTxQueue(queueCfg)
		if err != nil {
			return err
		}
		allConnections = append(allConnections, txPublisher)

		oldBlockWorkers[i] = parser.NewBlockWorker(ctx, blockConsumer, txPublisher, int(i), "old")
	}

	// Old tx workers: consume from old-tx-queue
	oldTxWorkers := make([]parser.TxWorker, cfg.OldTxWorkers)
	for i := range oldTxWorkers {
		txConsumer, err := queue.ConnectOldTxQueue(queueCfg)
		if err != nil {
			return err
		}
		allConnections = append(allConnections, txConsumer)
		oldTxWorkers[i] = parser.NewTxWorker(ctx, txConsumer, int(i), "old")
	}

	waitGroup.Add(1)

	// Run all the async operations
	for _, module := range ctx.Modules {
		if module, ok := module.(modules.AsyncOperationsModule); ok {
			go module.RunAsyncOperations()
		}
	}

	// Start new block pipeline workers
	for i, w := range newBlockWorkers {
		ctx.Logger.Debug("starting new block worker...", "number", i+1)
		go w.Start()
	}
	for i, w := range newTxWorkers {
		ctx.Logger.Debug("starting new tx worker...", "number", i+1)
		go w.Start()
	}

	// Start old block pipeline workers
	for i, w := range oldBlockWorkers {
		ctx.Logger.Debug("starting old block worker...", "number", i+1)
		go w.Start()
	}
	for i, w := range oldTxWorkers {
		ctx.Logger.Debug("starting old tx worker...", "number", i+1)
		go w.Start()
	}

	// Listen for and trap any OS signal to gracefully shutdown and exit
	trapSignal(ctx, allConnections)

	if cfg.ParseGenesis {
		// Add the genesis to the old queue (it's historical data)
		if err := oldBlockPublisher.Publish(0); err != nil {
			return err
		}
	}

	if cfg.ParseOldBlocks {
		go enqueueMissingBlocks(oldBlockPublisher, ctx)
	}

	if cfg.ParseNewBlocks {
		go enqueueNewBlocks(newBlockPublisher, ctx)
	}

	// Block main process (signal capture will call WaitGroup's Done)
	waitGroup.Wait()
	return nil
}

// enqueueMissingBlocks enqueues jobs (block heights) for missed blocks starting
// at the startHeight up until the latest known height.
func enqueueMissingBlocks(exportQueue types.HeightQueue, ctx *parser.Context) {
	// Get the config
	cfg := config.Cfg.Parser

	// Get the latest height from RPC
	latestBlockHeight := mustGetLatestHeight(ctx)

	lastDbBlockHeight, err := ctx.Database.GetLastBlockHeight()
	fmt.Printf("[enqueueMissingBlocks] lastDbBlockHeight: %d\n", lastDbBlockHeight)

	if err != nil {
		ctx.Logger.Error("failed to get last block height from database", "error", err)
	}

	// Get the start height, default to the config's height
	startHeight := cfg.StartHeight

	// Set startHeight to the latest height in database
	// if is not set inside config.yaml file
	if startHeight == 0 {
		startHeight = utils.MaxInt64(1, lastDbBlockHeight)
	}

	if cfg.FastSync {
		ctx.Logger.Info("fast sync is enabled, ignoring all previous blocks", "latest_block_height", latestBlockHeight)
		for _, module := range ctx.Modules {
			if mod, ok := module.(modules.FastSyncModule); ok {
				err := mod.DownloadState(latestBlockHeight)
				if err != nil {
					ctx.Logger.Error("error while performing fast sync",
						"err", err,
						"last_block_height", latestBlockHeight,
						"module", module.Name(),
					)
				}
			}
		}
	} else {
		ctx.Logger.Info("[enqueueMissingBlocks] syncing missing blocks...", "latest_block_height", latestBlockHeight, "start_height", startHeight)
		for _, i := range ctx.Database.GetMissingHeights(startHeight, latestBlockHeight) {
			// ctx.Logger.Debug("[enqueueMissingBlocks] enqueueing missing block", "height", i)
			if err := exportQueue.Publish(i); err != nil {
				ctx.Logger.Error("failed to publish missing block", "height", i, "err", err)
			}
			time.Sleep(5 * time.Second)
		}
	}
}

// enqueueNewBlocks enqueues new block heights onto the provided queue.
func enqueueNewBlocks(exportQueue types.HeightQueue, ctx *parser.Context) {
	currHeight := mustGetLatestHeight(ctx)

	// Enqueue upcoming heights
	for {
		latestBlockHeight := mustGetLatestHeight(ctx)

		// Enqueue all heights from the current height up to the latest height
		for ; currHeight <= latestBlockHeight; currHeight++ {
			ctx.Logger.Debug("enqueueing new block", "height", currHeight)
			if err := exportQueue.Publish(currHeight); err != nil {
				ctx.Logger.Error("failed to publish new block", "height", currHeight, "err", err)
			}
		}
		time.Sleep(config.GetAvgBlockTime())
	}
}

// mustGetLatestHeight tries getting the latest height from the RPC client.
// If after 50 tries no latest height can be found, it returns 0.
func mustGetLatestHeight(ctx *parser.Context) int64 {
	for retryCount := 0; retryCount < 50; retryCount++ {
		latestBlockHeight, err := ctx.Node.LatestHeight()
		if err == nil {
			return latestBlockHeight
		}

		ctx.Logger.Error("failed to get last block from RPCConfig client",
			"err", err,
			"retry interval", config.GetAvgBlockTime(),
			"retry count", retryCount)

		time.Sleep(config.GetAvgBlockTime())
	}

	return 0
}

// trapSignal will listen for any OS signal and invoke Done on the main
// WaitGroup allowing the main process to gracefully exit.
func trapSignal(ctx *parser.Context, connections []io.Closer) {
	var sigCh = make(chan os.Signal, 1)

	signal.Notify(sigCh, syscall.SIGTERM)
	signal.Notify(sigCh, syscall.SIGINT)

	go func() {
		sig := <-sigCh
		ctx.Logger.Info("caught signal; shutting down...", "signal", sig.String())
		for _, conn := range connections {
			_ = conn.Close()
		}
		defer ctx.Node.Stop()
		defer ctx.Database.Close()
		defer waitGroup.Done()
	}()
}
