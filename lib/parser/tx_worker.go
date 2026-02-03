package parser

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/1119-Labs/callisto/v4/lib/database"
	"github.com/1119-Labs/callisto/v4/lib/logging"
	"github.com/1119-Labs/callisto/v4/lib/modules"
	"github.com/1119-Labs/callisto/v4/lib/node"
	"github.com/1119-Labs/callisto/v4/lib/types"
	"github.com/1119-Labs/callisto/v4/lib/types/config"
)

// TxWorker processes transaction hashes from the tx queue.
// It fetches transaction details via gRPC API and saves to the database.
type TxWorker struct {
	index        int
	pipelineType string // "new" or "old"

	txQueue types.TxQueue
	modules []modules.Module

	node   node.Node
	db     database.Database
	logger logging.Logger
}

// NewTxWorker creates a new TxWorker instance.
// pipelineType should be "new" for real-time txs or "old" for backfill/missing txs.
func NewTxWorker(ctx *Context, txQueue types.TxQueue, index int, pipelineType string) TxWorker {
	return TxWorker{
		index:        index,
		pipelineType: pipelineType,
		node:         ctx.Node,
		txQueue:      txQueue,
		db:           ctx.Database,
		modules:      ctx.Modules,
		logger:       ctx.Logger,
	}
}

// Start starts the tx worker by consuming from the tx queue.
func (w TxWorker) Start() {
	if w.txQueue == nil {
		w.logger.Error("tx worker queue is nil")
		return
	}

	fmt.Printf("[TxWorker-%d] Starting tx worker\n", w.index)

	logging.WorkerCount.Inc()
	chainID, err := w.node.ChainID()
	if err != nil {
		w.logger.Error("error while getting chain ID from the node", "err", err)
	}

	fmt.Printf("[TxWorker-%d] Waiting for messages from tx queue...\n", w.index)

	err = w.txQueue.Consume(func(txHash string, height int64) error {
		fmt.Printf("[TxWorker-%d] Received tx from queue: hash=%s, height=%d\n", w.index, txHash, height)

		if err := w.ProcessTx(txHash, height); err != nil {
			time.Sleep(config.GetAvgBlockTime())
			w.logger.Error("re-enqueueing failed tx", "tx_hash", txHash, "height", height, "err", err)
			return err
		}

		logging.WorkerHeight.WithLabelValues(fmt.Sprintf("tx-%d", w.index), chainID).Set(float64(height))
		return nil
	})
	if err != nil {
		w.logger.Error("tx worker consume failed", "err", err)
	}
}

// ProcessTx fetches a transaction by hash and processes it.
func (w TxWorker) ProcessTx(txHash string, height int64) error {
	w.logger.Debug("processing tx", "tx_hash", txHash, "height", height)

	// Fetch the transaction from the node
	tx, err := w.node.Tx(txHash)
	if err != nil {
		return fmt.Errorf("failed to get tx from node: %s", err)
	}

	// Save the transaction
	if err := w.saveTx(tx); err != nil {
		return err
	}

	// Call tx handlers
	w.handleTx(tx)

	// Call message handlers
	for i, msg := range tx.Tx.Body.Messages {
		w.handleMessage(i, msg, tx)
	}

	// Update metrics
	totalBlocks := w.db.GetTotalBlocks()
	logging.DbBlockCount.WithLabelValues("total_blocks_in_db").Set(float64(totalBlocks))

	dbLatestHeight, err := w.db.GetLastBlockHeight()
	if err != nil {
		return err
	}
	logging.DbLatestHeight.WithLabelValues("db_latest_height").Set(float64(dbLatestHeight))

	return nil
}

// saveTx persists a transaction to the database.
func (w TxWorker) saveTx(tx *types.Transaction) error {
	err := w.db.SaveTx(tx)
	if err != nil {
		return fmt.Errorf("failed to handle transaction with hash %s: %s", tx.TxResponse.TxHash, err)
	}
	return nil
}

// handleTx calls all registered transaction handlers.
func (w TxWorker) handleTx(tx *types.Transaction) {
	for _, module := range w.modules {
		if transactionModule, ok := module.(modules.TransactionModule); ok {
			err := transactionModule.HandleTx(tx)
			if err != nil {
				w.logger.TxError(module, tx, err)
			}
		}
	}
}

// handleMessage handles a single message within a transaction.
func (w TxWorker) handleMessage(index int, msg types.Message, tx *types.Transaction) {
	// Allow modules to handle the message
	for _, module := range w.modules {
		if messageModule, ok := module.(modules.MessageModule); ok {
			err := messageModule.HandleMsg(index, msg, tx)
			if err != nil {
				w.logger.MsgError(module, tx, msg, err)
			}
		}

		// If it's a MsgExecute, handle inner messages
		if msg.GetType() == "/cosmos.authz.v1beta1.MsgExec" {
			var msgExec struct {
				Msgs []json.RawMessage `json:"msgs"`
			}

			err := json.Unmarshal(msg.GetBytes(), &msgExec)
			if err != nil {
				w.logger.Error("unable to unmarshal MsgExec inner messages", "error", err)
				return
			}

			for authzIndex, msgAny := range msgExec.Msgs {
				executedMsg, err := types.UnmarshalMessage(authzIndex, msgAny)
				if err != nil {
					w.logger.Error("unable to unpack MsgExec inner message", "index", authzIndex, "error", err)
				}

				for _, module := range w.modules {
					if messageModule, ok := module.(modules.AuthzMessageModule); ok {
						err = messageModule.HandleMsgExec(index, authzIndex, executedMsg, tx)
						if err != nil {
							w.logger.MsgError(module, tx, executedMsg, err)
						}
					}
				}
			}
		}
	}
}
