package parser

import (
	"encoding/json"
	"fmt"

	"github.com/1119-Labs/callisto/v4/lib/database"
	"github.com/1119-Labs/callisto/v4/lib/logging"
	"github.com/1119-Labs/callisto/v4/lib/modules"
	"github.com/1119-Labs/callisto/v4/lib/node"
	"github.com/1119-Labs/callisto/v4/lib/types"
	"github.com/1119-Labs/callisto/v4/lib/types/config"
	"github.com/1119-Labs/callisto/v4/lib/types/utils"

	tmctypes "github.com/cometbft/cometbft/rpc/core/types"
	tmtypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BlockWorker processes block heights from the block queue.
// It fetches all block data (including transactions) via a single API call and saves to DB.
type BlockWorker struct {
	index        int
	pipelineType string // "new" or "old"

	blockQueue       types.HeightQueue
	modules          []modules.Module
	addressExtractor *AddressExtractor

	node   node.Node
	db     database.Database
	logger logging.Logger
}

// NewBlockWorker creates a new BlockWorker instance.
// pipelineType should be "new" for real-time blocks or "old" for backfill/missing blocks.
func NewBlockWorker(ctx *Context, blockQueue types.HeightQueue, index int, pipelineType string) BlockWorker {
	return BlockWorker{
		index:            index,
		pipelineType:     pipelineType,
		node:             ctx.Node,
		blockQueue:       blockQueue,
		db:               ctx.Database,
		modules:          ctx.Modules,
		logger:           ctx.Logger,
		addressExtractor: NewAddressExtractor(),
	}
}

// Start starts the block worker by consuming from the block queue.
func (w BlockWorker) Start() {
	if w.blockQueue == nil {
		w.logger.Error(fmt.Sprintf("[BlockWorker-%s-%d] block worker queue is nil", w.pipelineType, w.index))
		return
	}

	logging.WorkerCount.Inc()
	chainID, err := w.node.ChainID()
	if err != nil {
		w.logger.Error(fmt.Sprintf("[BlockWorker-%s-%d] error while getting chain ID from the node", w.pipelineType, w.index), "err", err)
	}

	err = w.blockQueue.Consume(func(height int64) error {
		if err := w.ProcessIfNotExists(height); err != nil {
			w.logger.Error(fmt.Sprintf("[BlockWorker-%s-%d] failed to process block", w.pipelineType, w.index), "height", height, "err", err)
			return err
		}

		logging.WorkerHeight.WithLabelValues(fmt.Sprintf("%d", w.index), chainID).Set(float64(height))
		return nil
	})
	if err != nil {
		w.logger.Error(fmt.Sprintf("[BlockWorker-%s-%d] block worker consume failed", w.pipelineType, w.index), "err", err)
	}
}

// StartDLQ starts a dead-letter queue worker that force-retries failed blocks.
// Unlike Start(), it always calls Process() (never skips already-existing blocks)
// since the whole point is to retry something that previously failed.
func (w BlockWorker) StartDLQ() {
	if w.blockQueue == nil {
		w.logger.Error(fmt.Sprintf("[BlockWorker-%s-%d] DLQ worker queue is nil", w.pipelineType, w.index))
		return
	}

	logging.WorkerCount.Inc()
	chainID, err := w.node.ChainID()
	if err != nil {
		w.logger.Error(fmt.Sprintf("[BlockWorker-%s-%d] error while getting chain ID from the node", w.pipelineType, w.index), "err", err)
	}

	w.logger.Info(fmt.Sprintf("[BlockWorker-%s-%d] DLQ retry worker started", w.pipelineType, w.index))

	err = w.blockQueue.Consume(func(height int64) error {
		w.logger.Info(fmt.Sprintf("[BlockWorker-%s-%d] DLQ retrying block", w.pipelineType, w.index), "height", height)

		if err := w.Process(height); err != nil {
			w.logger.Error(fmt.Sprintf("[BlockWorker-%s-%d] DLQ retry failed for block", w.pipelineType, w.index), "height", height, "err", err)
			return err
		}

		w.logger.Info(fmt.Sprintf("[BlockWorker-%s-%d] DLQ retry succeeded for block", w.pipelineType, w.index), "height", height)
		logging.WorkerHeight.WithLabelValues(fmt.Sprintf("dlq-%d", w.index), chainID).Set(float64(height))
		return nil
	})
	if err != nil {
		w.logger.Error(fmt.Sprintf("[BlockWorker-%s-%d] DLQ worker consume failed", w.pipelineType, w.index), "err", err)
	}
}

// ProcessIfNotExists processes a block if it doesn't already exist in the database.
func (w BlockWorker) ProcessIfNotExists(height int64) error {
	exists, err := w.db.HasBlock(height)
	if err != nil {
		return fmt.Errorf("error while searching for block: %s", err)
	}

	if exists {
		w.logger.Debug(fmt.Sprintf("[BlockWorker-%s-%d] skipping already exported block", w.pipelineType, w.index), "height", height)
		return nil
	}

	return w.Process(height)
}

// Process fetches a block and all its transactions via API, then saves everything to DB.
func (w BlockWorker) Process(height int64) error {
	if height == 0 {
		cfg := config.Cfg.Parser
		genesisDoc, genesisState, err := utils.GetGenesisDocAndState(cfg.GenesisFilePath, w.node)
		if err != nil {
			return fmt.Errorf("failed to get genesis: %s", err)
		}
		return w.HandleGenesis(genesisDoc, genesisState)
	}

	// Fetch block data
	block, err := w.node.Block(height)
	if err != nil {
		return fmt.Errorf("failed to get block from node: %s", err)
	}

	events, err := w.node.BlockResults(height)
	if err != nil {
		return fmt.Errorf("failed to get block results from node: %s", err)
	}

	vals, err := w.node.Validators(height)
	if err != nil {
		return fmt.Errorf("failed to get validators for block: %s", err)
	}

	// Fetch all transactions for this block via the dedicated API
	txs, err := w.node.BlockTransactions(height)
	if err != nil {
		w.logger.Error(fmt.Sprintf("[BlockWorker-%s-%d] failed to fetch block transactions", w.pipelineType, w.index), "height", height, "err", err)
		// Continue with block processing even if tx fetch fails
		txs = nil
	}

	// Export block and consensus data
	if err := w.ExportBlock(block, events, vals); err != nil {
		return err
	}

	// Process all transactions (skip first tx in block - it's a system tx)
	if len(txs) > 1 {
		for i := 1; i < len(txs); i++ {
			tx := txs[i]
			if err := w.ProcessTx(tx); err != nil {
				w.logger.Error(fmt.Sprintf("[BlockWorker-%s-%d] failed to process tx", w.pipelineType, w.index), "tx_hash", tx.TxHash, "height", height, "err", err)
				// Continue processing other transactions
			}
		}
	}

	// Update metrics
	totalBlocks := w.db.GetTotalBlocks()
	logging.DbBlockCount.WithLabelValues("total_blocks_in_db").Set(float64(totalBlocks))

	dbLatestHeight, err := w.db.GetLastBlockHeight()
	if err == nil {
		logging.DbLatestHeight.WithLabelValues("db_latest_height").Set(float64(dbLatestHeight))
	}

	return nil
}

// HandleGenesis handles genesis block processing.
func (w BlockWorker) HandleGenesis(genesisDoc *tmtypes.GenesisDoc, appState map[string]json.RawMessage) error {
	for _, module := range w.modules {
		if genesisModule, ok := module.(modules.GenesisModule); ok {
			if err := genesisModule.HandleGenesis(genesisDoc, appState); err != nil {
				w.logger.GenesisError(module, err)
			}
		}
	}
	return nil
}

// ExportBlock exports block and consensus data to the database.
func (w BlockWorker) ExportBlock(
	b *tmctypes.ResultBlock, r *tmctypes.ResultBlockResults, vals *tmctypes.ResultValidators,
) error {
	// Save all validators
	if err := w.SaveValidators(vals.Validators); err != nil {
		return err
	}

	// Make sure the proposer exists
	proposerAddr := sdk.ConsAddress(b.Block.ProposerAddress)
	val := findValidatorByAddr(proposerAddr.String(), vals)
	if val == nil {
		return fmt.Errorf("failed to find validator by proposer address %s", proposerAddr.String())
	}

	// Save the block (with 0 gas for now, tx worker will update)
	err := w.db.SaveBlock(types.NewBlockFromTmBlock(b, 0))
	if err != nil {
		return fmt.Errorf("failed to persist block: %s", err)
	}

	// Save the commits
	if err := w.ExportCommit(b.Block.LastCommit, vals); err != nil {
		return err
	}

	// Call the block handlers (without txs)
	for _, module := range w.modules {
		if blockModule, ok := module.(modules.BlockModule); ok {
			err = blockModule.HandleBlock(b, r, nil, vals)
			if err != nil {
				w.logger.BlockError(module, b, err)
			}
		}
	}

	return nil
}

// SaveValidators persists validators to the database.
func (w BlockWorker) SaveValidators(vals []*tmtypes.Validator) error {
	var validators = make([]*types.Validator, len(vals))
	for index, val := range vals {
		consAddr := sdk.ConsAddress(val.Address).String()

		consPubKey, err := types.ConvertValidatorPubKeyToBech32String(val.PubKey)
		if err != nil {
			return fmt.Errorf("failed to convert validator public key for validators %s: %s", consAddr, err)
		}

		validators[index] = types.NewValidator(consAddr, consPubKey)
	}

	err := w.db.SaveValidators(validators)
	if err != nil {
		return fmt.Errorf("error while saving validators: %s", err)
	}

	return nil
}

// ExportCommit exports commit signatures to the database.
func (w BlockWorker) ExportCommit(commit *tmtypes.Commit, vals *tmctypes.ResultValidators) error {
	var signatures []*types.CommitSig
	for _, commitSig := range commit.Signatures {
		if commitSig.Signature == nil {
			continue
		}

		valAddr := sdk.ConsAddress(commitSig.ValidatorAddress)
		val := findValidatorByAddr(valAddr.String(), vals)
		if val == nil {
			return fmt.Errorf("failed to find validator by commit validator address %s", valAddr.String())
		}

		signatures = append(signatures, types.NewCommitSig(
			types.ConvertValidatorAddressToBech32String(commitSig.ValidatorAddress),
			val.VotingPower,
			val.ProposerPriority,
			commit.Height,
			commitSig.Timestamp,
		))
	}

	err := w.db.SaveCommitSignatures(signatures)
	if err != nil {
		return fmt.Errorf("error while saving commit signatures: %s", err)
	}

	return nil
}

// ProcessTx processes a single transaction - saves it and all related data to DB.
func (w BlockWorker) ProcessTx(tx *types.Transaction) error {
	if tx == nil {
		return nil
	}

	w.logger.Debug(fmt.Sprintf("[BlockWorker-%s-%d] processing tx", w.pipelineType, w.index), "tx_hash", tx.TxHash, "height", tx.Height)

	// Extract all involved accounts from the transaction using the registry-based extractor
	accounts := w.addressExtractor.ExtractFromTx(tx)

	// Save accounts first (they need to exist before we can reference them)
	if err := w.db.SaveAccounts(accounts); err != nil {
		w.logger.Error(fmt.Sprintf("[BlockWorker-%s-%d] failed to save accounts", w.pipelineType, w.index), "tx_hash", tx.TxHash, "err", err)
		// Don't fail the transaction processing for account saving errors
	}

	// Save the transaction
	if err := w.saveTx(tx); err != nil {
		return err
	}

	// Save transaction-account relationships
	if err := w.db.SaveTxAccounts(tx.TxHash, int64(tx.Height), accounts); err != nil {
		w.logger.Error(fmt.Sprintf("[BlockWorker-%s-%d] failed to save tx-account relationships", w.pipelineType, w.index), "tx_hash", tx.TxHash, "err", err)
		// Don't fail the transaction processing for relationship saving errors
	}

	// Call tx handlers
	w.handleTx(tx)

	// Call message handlers (only if tx body is available)
	if tx.Tx != nil && tx.Tx.Body != nil {
		for i, msg := range tx.Tx.Body.Messages {
			w.handleMessage(i, msg, tx)
		}
	}

	return nil
}

// saveTx persists a transaction to the database.
func (w BlockWorker) saveTx(tx *types.Transaction) error {
	if tx == nil {
		return nil
	}
	err := w.db.SaveTx(tx)
	if err != nil {
		txHash := ""
		if tx.TxResponse != nil {
			txHash = tx.TxResponse.TxHash
		}
		return fmt.Errorf("failed to handle transaction with hash %s: %s", txHash, err)
	}
	return nil
}

// handleTx calls all registered transaction handlers.
func (w BlockWorker) handleTx(tx *types.Transaction) {
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
func (w BlockWorker) handleMessage(index int, msg types.Message, tx *types.Transaction) {
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
				w.logger.Error(fmt.Sprintf("[BlockWorker-%s-%d] unable to unmarshal MsgExec inner messages", w.pipelineType, w.index), "error", err)
				return
			}

			for authzIndex, msgAny := range msgExec.Msgs {
				executedMsg, err := types.UnmarshalMessage(authzIndex, msgAny)
				if err != nil {
					w.logger.Error(fmt.Sprintf("[BlockWorker-%s-%d] unable to unpack MsgExec inner message", w.pipelineType, w.index), "index", authzIndex, "error", err)
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
