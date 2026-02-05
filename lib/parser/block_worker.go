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
	"github.com/1119-Labs/callisto/v4/lib/types/utils"

	tmctypes "github.com/cometbft/cometbft/rpc/core/types"
	tmtypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BlockWorker processes block heights from the block queue.
// It fetches block data via RPC, saves block/consensus info to DB,
// and publishes transaction hashes to the tx queue.
type BlockWorker struct {
	index        int
	pipelineType string // "new" or "old"

	blockQueue types.HeightQueue
	txQueue    types.TxQueue
	modules    []modules.Module

	node   node.Node
	db     database.Database
	logger logging.Logger
}

// NewBlockWorker creates a new BlockWorker instance.
// pipelineType should be "new" for real-time blocks or "old" for backfill/missing blocks.
func NewBlockWorker(ctx *Context, blockQueue types.HeightQueue, txQueue types.TxQueue, index int, pipelineType string) BlockWorker {
	return BlockWorker{
		index:        index,
		pipelineType: pipelineType,
		node:         ctx.Node,
		blockQueue:   blockQueue,
		txQueue:      txQueue,
		db:           ctx.Database,
		modules:      ctx.Modules,
		logger:       ctx.Logger,
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
			time.Sleep(config.GetAvgBlockTime())
			w.logger.Error(fmt.Sprintf("[BlockWorker-%s-%d] re-enqueueing failed block", w.pipelineType, w.index), "height", height, "err", err)
			return err
		}

		logging.WorkerHeight.WithLabelValues(fmt.Sprintf("%d", w.index), chainID).Set(float64(height))
		return nil
	})
	if err != nil {
		w.logger.Error(fmt.Sprintf("[BlockWorker-%s-%d] block worker consume failed", w.pipelineType, w.index), "err", err)
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

// Process fetches a block and exports block/consensus data, then publishes tx hashes to tx queue.
func (w BlockWorker) Process(height int64) error {
	if height == 0 {
		cfg := config.Cfg.Parser
		genesisDoc, genesisState, err := utils.GetGenesisDocAndState(cfg.GenesisFilePath, w.node)
		if err != nil {
			return fmt.Errorf("failed to get genesis: %s", err)
		}
		return w.HandleGenesis(genesisDoc, genesisState)
	}

	// w.logger.Debug(fmt.Sprintf("[BlockWorker-%s-%d] processing block", w.pipelineType, w.index), "height", height)

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

	// Publish tx hashes to tx queue FIRST (so tx workers can start processing early)
	// This allows tx workers to fetch tx details in parallel while we save block data
	// fmt.Printf("[BlockWorker-%s-%d] Block %d has %d transactions\n", w.pipelineType, w.index, height, len(block.Block.Txs))
	for i, tx := range block.Block.Txs {
		if i > 0 {
			txHash := fmt.Sprintf("%X", tx.Hash())
			// fmt.Printf("[BlockWorker-%s-%d] Publishing tx to queue: hash=%s, height=%d\n", w.pipelineType, w.index, txHash, height)
			if err := w.txQueue.Publish(txHash, height); err != nil {
				w.logger.Error(fmt.Sprintf("[BlockWorker-%s-%d] failed to publish tx to queue", w.pipelineType, w.index), "tx_hash", txHash, "height", height, "err", err)
			}
		}

	}

	// Export block and consensus data (DB write - can take time)
	if err := w.ExportBlock(block, events, vals); err != nil {
		return err
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
