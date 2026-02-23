package postgresql

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/1119-Labs/callisto/v4/lib/logging"

	"github.com/lib/pq"

	"github.com/1119-Labs/callisto/v4/lib/database"
	"github.com/1119-Labs/callisto/v4/lib/types"
	"github.com/1119-Labs/callisto/v4/lib/types/config"
	"github.com/1119-Labs/callisto/v4/lib/types/env"
	"github.com/1119-Labs/callisto/v4/lib/types/utils"
)

// Builder creates a database connection with the given database connection info
// from config. It returns a database connection handle or an error if the
// connection fails.
func Builder(ctx *database.Context) (database.Database, error) {
	dbURI := utils.GetEnvOr(env.DatabaseURI, ctx.Cfg.URL)
	dbEnableSSL := utils.GetEnvOr(env.DatabaseSSLModeEnable, ctx.Cfg.SSLModeEnable)

	// Configure SSL certificates (optional)
	if dbEnableSSL == "true" {
		dbRootCert := utils.GetEnvOr(env.DatabaseSSLRootCert, ctx.Cfg.SSLRootCert)
		dbCert := utils.GetEnvOr(env.DatabaseSSLCert, ctx.Cfg.SSLCert)
		dbKey := utils.GetEnvOr(env.DatabaseSSLKey, ctx.Cfg.SSLKey)
		dbURI += fmt.Sprintf(" sslmode=require sslrootcert=%s sslcert=%s sslkey=%s",
			dbRootCert, dbCert, dbKey)
	}

	postgresDb, err := sqlx.Open("postgres", dbURI)
	if err != nil {
		return nil, err
	}

	// Configure connection pool
	postgresDb.SetMaxOpenConns(ctx.Cfg.MaxOpenConnections)
	postgresDb.SetMaxIdleConns(ctx.Cfg.MaxIdleConnections)
	if ctx.Cfg.ConnMaxLifetimeSeconds > 0 {
		postgresDb.SetConnMaxLifetime(time.Duration(ctx.Cfg.ConnMaxLifetimeSeconds) * time.Second)
	}

	return &Database{
		SQL:    postgresDb,
		Logger: ctx.Logger,
	}, nil
}

// type check to ensure interface is properly implemented
var _ database.Database = &Database{}

// Database defines a wrapper around a SQL database and implements functionality
// for data aggregation and exporting.
type Database struct {
	SQL    *sqlx.DB
	Logger logging.Logger
}

// // CreatePartitionIfNotExists creates a new partition having the given partition id if not existing
// func (db *Database) CreatePartitionIfNotExists(table string, partitionID int64) error {
// 	partitionTable := fmt.Sprintf("%s_%d", table, partitionID)

// 	stmt := fmt.Sprintf(
// 		"CREATE TABLE IF NOT EXISTS %s PARTITION OF %s FOR VALUES IN (%d)",
// 		partitionTable,
// 		table,
// 		partitionID,
// 	)
// 	_, err := db.SQL.Exec(stmt)

// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

func (db *Database) CreatePartitionIfNotExists(table string, partitionID int64) error {
	partitionTable := fmt.Sprintf("%s_%d", table, partitionID)

	// Check if partition already exists
	var exists bool
	checkStmt := `
  SELECT EXISTS (
   SELECT 1 FROM pg_tables 
   WHERE tablename = $1
  )
 `
	err := db.SQL.QueryRow(checkStmt, partitionTable).Scan(&exists)
	if err != nil {
		return err
	}

	// If partition exists, skip creation
	if exists {
		return nil
	}

	// Create the partition
	stmt := fmt.Sprintf(
		"CREATE TABLE %s PARTITION OF %s FOR VALUES IN (%d)",
		partitionTable,
		table,
		partitionID,
	)
	_, err = db.SQL.Exec(stmt)

	if err != nil {
		return err
	}

	return nil
}

// -------------------------------------------------------------------------------------------------------------------

// HasBlock implements database.Database
func (db *Database) HasBlock(height int64) (bool, error) {
	var res bool
	err := db.SQL.QueryRow(`SELECT EXISTS(SELECT 1 FROM block WHERE height = $1);`, height).Scan(&res)
	return res, err
}

// GetLastBlockHeight returns the last block height stored inside the database
func (db *Database) GetLastBlockHeight() (int64, error) {
	stmt := `SELECT height FROM block ORDER BY height DESC LIMIT 1;`

	var height int64
	err := db.SQL.QueryRow(stmt).Scan(&height)
	if err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			// If no rows stored in block table, return 0 as height
			return 0, nil
		}
		return 0, fmt.Errorf("error while getting last block height, error: %s", err)
	}

	return height, nil
}

// GetMissingHeights returns a slice of missing block heights between startHeight and endHeight
func (db *Database) GetMissingHeights(startHeight, endHeight int64) []int64 {
	var result []int64
	stmt := `SELECT generate_series($1::int,$2::int) EXCEPT SELECT height FROM block ORDER BY 1;`
	err := db.SQL.Select(&result, stmt, startHeight, endHeight)
	if err != nil {
		return nil
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

// SaveBlock implements database.Database
func (db *Database) SaveBlock(block *types.Block) error {
	sqlStatement := `
INSERT INTO block (height, hash, num_txs, total_gas, proposer_address, timestamp)
VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT DO NOTHING`

	proposerAddress := sql.NullString{Valid: len(block.ProposerAddress) != 0, String: block.ProposerAddress}
	_, err := db.SQL.Exec(sqlStatement,
		block.Height, block.Hash, block.TxNum, block.TotalGas, proposerAddress, block.Timestamp,
	)
	return err
}

// GetTotalBlocks implements database.Database
func (db *Database) GetTotalBlocks() int64 {
	var blockCount int64
	err := db.SQL.QueryRow(`SELECT count(*) FROM block;`).Scan(&blockCount)
	if err != nil {
		return 0
	}

	return blockCount
}

// SaveTx implements database.Database
func (db *Database) SaveTx(tx *types.Transaction) error {

	if tx == nil {
		return fmt.Errorf("error nil transaction")
	}

	var partitionID int64

	partitionSize := config.Cfg.Database.PartitionSize
	if partitionSize > 0 {
		partitionID = int64(tx.Height) / partitionSize
		err := db.CreatePartitionIfNotExists("transaction", partitionID)
		if err != nil {
			return err
		}
	}

	return db.saveTxInsidePartition(tx, partitionID)
}

// saveTxInsidePartition stores the given transaction inside the partition having the given id
func (db *Database) saveTxInsidePartition(tx *types.Transaction, partitionID int64) error {
	sqlStatement := `
INSERT INTO transaction 
(hash, height, success, messages, memo, signatures, signer_infos, fee, gas_wanted, gas_used, raw_log, logs,
 code, codespace, data, info, timestamp, events, timeout_height, extension_options, non_critical_extension_options, tip,
 raw_json, partition_id) 
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24) 
ON CONFLICT (hash, height, partition_id) DO UPDATE 
	SET success = excluded.success, 
		messages = excluded.messages,
		memo = excluded.memo, 
		signatures = excluded.signatures, 
		signer_infos = excluded.signer_infos,
		fee = excluded.fee, 
		gas_wanted = excluded.gas_wanted, 
		gas_used = excluded.gas_used,
		raw_log = excluded.raw_log, 
		logs = excluded.logs,
		code = excluded.code,
		codespace = excluded.codespace,
		data = excluded.data,
		info = excluded.info,
		timestamp = excluded.timestamp,
		events = excluded.events,
		timeout_height = excluded.timeout_height,
		extension_options = excluded.extension_options,
		non_critical_extension_options = excluded.non_critical_extension_options,
		tip = excluded.tip,
		raw_json = excluded.raw_json`

	// Extract body fields (handle nil tx body for failed txs where tx is null)
	var sigs []string
	var msgsBz string
	var memo string
	var feeBz []byte
	var sigInfoBz string
	var timeoutHeight uint64
	var extensionOptsBz []byte
	var nonCriticalExtOptsBz []byte
	var tipBz []byte

	if tx.Tx != nil && tx.Tx.Body != nil {
		sigs = make([]string, len(tx.Signatures))
		for index, sig := range tx.Signatures {
			sigs[index] = base64.StdEncoding.EncodeToString(sig)
		}

		msgs := make([]string, len(tx.Body.Messages))
		for index, msg := range tx.Body.Messages {
			msgs[index] = string(msg.GetBytes())
		}
		msgsBz = fmt.Sprintf("[%s]", strings.Join(msgs, ","))
		memo = tx.Body.Memo
		timeoutHeight = tx.Body.TimeoutHeight

		// Extension options from body
		if tx.Body.TxBody != nil && len(tx.Body.TxBody.ExtensionOptions) > 0 {
			extensionOptsBz, _ = json.Marshal(tx.Body.TxBody.ExtensionOptions)
		}
		if tx.Body.TxBody != nil && len(tx.Body.TxBody.NonCriticalExtensionOptions) > 0 {
			nonCriticalExtOptsBz, _ = json.Marshal(tx.Body.TxBody.NonCriticalExtensionOptions)
		}
	} else {
		sigs = []string{}
		msgsBz = "[]"
	}

	if tx.Tx != nil && tx.Tx.AuthInfo != nil {
		var err error
		feeBz, err = json.Marshal(tx.AuthInfo.Fee)
		if err != nil {
			return fmt.Errorf("failed to JSON encode tx fee: %s", err)
		}

		sigInfos := make([]string, len(tx.AuthInfo.SignerInfos))
		for index, info := range tx.AuthInfo.SignerInfos {
			bz, err := json.Marshal(info)
			if err != nil {
				return err
			}
			sigInfos[index] = string(bz)
		}
		sigInfoBz = fmt.Sprintf("[%s]", strings.Join(sigInfos, ","))

		// Tip (if available)
		if tx.AuthInfo.AuthInfo != nil && tx.AuthInfo.AuthInfo.Tip != nil {
			tipBz, _ = json.Marshal(tx.AuthInfo.AuthInfo.Tip)
		}
	} else {
		feeBz = []byte("{}")
		sigInfoBz = "[]"
	}

	logsBz, err := json.Marshal(tx.Logs)
	if err != nil {
		return err
	}

	// Extract TxResponse extra fields
	var code uint32
	var codespace string
	var txData string
	var info string
	var txTimestamp *time.Time
	var eventsBz []byte

	if tx.TxResponse != nil {
		code = tx.TxResponse.Code
		codespace = tx.TxResponse.Codespace
		txData = tx.TxResponse.Data
		info = tx.TxResponse.Info

		// Parse timestamp from TxResponse
		if tx.TxResponse.Timestamp != "" {
			t, parseErr := time.Parse(time.RFC3339Nano, tx.TxResponse.Timestamp)
			if parseErr == nil {
				txTimestamp = &t
			}
		}

		// Marshal events from the embedded sdk.TxResponse
		if tx.TxResponse.TxResponse != nil && len(tx.TxResponse.TxResponse.Events) > 0 {
			eventsBz, _ = json.Marshal(tx.TxResponse.TxResponse.Events)
		}
	}

	// Build nullable JSON values for JSONB columns
	extensionOptsVal := toNullableJSONString(extensionOptsBz, "[]")
	nonCriticalExtOptsVal := toNullableJSONString(nonCriticalExtOptsBz, "[]")
	tipVal := toNullableJSONString(tipBz, "")
	eventsVal := toNullableJSONString(eventsBz, "")
	rawJSONVal := toNullableJSONString(tx.RawJSON, "")

	_, err = db.SQL.Exec(sqlStatement,
		tx.TxHash, tx.Height, tx.Successful(),
		msgsBz, memo, pq.Array(sigs),
		sigInfoBz, string(feeBz),
		tx.GasWanted, tx.GasUsed, tx.RawLog, string(logsBz),
		code, codespace, txData, info, txTimestamp, eventsVal,
		timeoutHeight, extensionOptsVal, nonCriticalExtOptsVal, tipVal,
		rawJSONVal, partitionID,
	)
	return err
}

// toNullableJSONString converts a byte slice to a sql.NullString suitable for JSONB columns.
// If bz is nil/empty, returns the fallback value (or NULL if fallback is empty).
func toNullableJSONString(bz []byte, fallback string) sql.NullString {
	if len(bz) > 0 {
		return sql.NullString{Valid: true, String: string(bz)}
	}
	if fallback != "" {
		return sql.NullString{Valid: true, String: fallback}
	}
	return sql.NullString{Valid: false}
}

// HasValidator implements database.Database
func (db *Database) HasValidator(addr string) (bool, error) {
	var res bool
	stmt := `SELECT EXISTS(SELECT 1 FROM validator WHERE consensus_address = $1);`
	err := db.SQL.QueryRow(stmt, addr).Scan(&res)
	return res, err
}

// SaveValidators implements database.Database
func (db *Database) SaveValidators(validators []*types.Validator) error {
	if len(validators) == 0 {
		return nil
	}

	stmt := `INSERT INTO validator (consensus_address, consensus_pubkey) VALUES `

	var vparams []interface{}
	for i, val := range validators {
		vi := i * 2

		stmt += fmt.Sprintf("($%d, $%d),", vi+1, vi+2)
		vparams = append(vparams, val.ConsAddr, val.ConsPubKey)
	}

	stmt = stmt[:len(stmt)-1] // Remove trailing ,
	stmt += " ON CONFLICT DO NOTHING"
	_, err := db.SQL.Exec(stmt, vparams...)
	return err
}

// SaveCommitSignatures implements database.Database
func (db *Database) SaveCommitSignatures(signatures []*types.CommitSig) error {
	if len(signatures) == 0 {
		return nil
	}

	stmt := `INSERT INTO pre_commit (validator_address, height, timestamp, voting_power, proposer_priority) VALUES `

	var sparams []interface{}
	for i, sig := range signatures {
		si := i * 5

		stmt += fmt.Sprintf("($%d, $%d, $%d, $%d, $%d),", si+1, si+2, si+3, si+4, si+5)
		sparams = append(sparams, sig.ValidatorAddress, sig.Height, sig.Timestamp, sig.VotingPower, sig.ProposerPriority)
	}

	stmt = stmt[:len(stmt)-1]
	stmt += " ON CONFLICT (validator_address, timestamp) DO NOTHING"
	_, err := db.SQL.Exec(stmt, sparams...)
	return err
}

// SaveMessage implements database.Database
func (db *Database) SaveMessage(height int64, txHash string, msg types.Message, addresses []string) error {
	var partitionID int64
	partitionSize := config.Cfg.Database.PartitionSize
	if partitionSize > 0 {
		partitionID = height / partitionSize
		err := db.CreatePartitionIfNotExists("message", partitionID)
		if err != nil {
			return err
		}
	}

	return db.saveMessageInsidePartition(height, txHash, addresses, msg, partitionID)
}

// saveMessageInsidePartition stores the given message inside the partition having the provided id
func (db *Database) saveMessageInsidePartition(height int64, txHash string, addresses []string, msg types.Message, partitionID int64) error {
	stmt := `
INSERT INTO message(transaction_hash, index, type, value, involved_accounts_addresses, height, partition_id) 
VALUES ($1, $2, $3, $4, $5, $6, $7) 
ON CONFLICT (transaction_hash, height, index, partition_id) DO UPDATE 
	SET type = excluded.type,
		value = excluded.value,
		involved_accounts_addresses = excluded.involved_accounts_addresses`

	_, err := db.SQL.Exec(stmt, txHash, msg.GetIndex(), msg.GetType(), msg.GetBytes(), pq.Array(addresses), height, partitionID)
	return err
}

// SaveAccounts implements database.Database
// It stores a list of account addresses if they do not already exist.
func (db *Database) SaveAccounts(addresses []string) error {
	if len(addresses) == 0 {
		return nil
	}

	// Remove duplicates
	uniqueAddresses := make(map[string]struct{})
	for _, addr := range addresses {
		if addr != "" {
			uniqueAddresses[addr] = struct{}{}
		}
	}

	if len(uniqueAddresses) == 0 {
		return nil
	}

	stmt := `INSERT INTO account (address) VALUES `
	var params []interface{}
	i := 0
	for addr := range uniqueAddresses {
		stmt += fmt.Sprintf("($%d),", i+1)
		params = append(params, addr)
		i++
	}

	stmt = stmt[:len(stmt)-1] // Remove trailing comma
	stmt += " ON CONFLICT DO NOTHING"

	_, err := db.SQL.Exec(stmt, params...)
	return err
}

// SaveTxAccounts implements database.Database
// It stores the relationship between a transaction and its involved accounts.
func (db *Database) SaveTxAccounts(txHash string, height int64, addresses []string) error {
	if len(addresses) == 0 {
		return nil
	}

	// Calculate partition ID
	var partitionID int64
	partitionSize := config.Cfg.Database.PartitionSize
	if partitionSize > 0 {
		partitionID = height / partitionSize
		err := db.CreatePartitionIfNotExists("transaction_account", partitionID)
		if err != nil {
			return err
		}
	}

	// Remove duplicates
	uniqueAddresses := make(map[string]struct{})
	for _, addr := range addresses {
		if addr != "" {
			uniqueAddresses[addr] = struct{}{}
		}
	}

	if len(uniqueAddresses) == 0 {
		return nil
	}

	stmt := `INSERT INTO transaction_account (transaction_hash, account_address, height, partition_id) VALUES `
	var params []interface{}
	i := 0
	for addr := range uniqueAddresses {
		paramOffset := i * 4
		stmt += fmt.Sprintf("($%d, $%d, $%d, $%d),", paramOffset+1, paramOffset+2, paramOffset+3, paramOffset+4)
		params = append(params, txHash, addr, height, partitionID)
		i++
	}

	stmt = stmt[:len(stmt)-1] // Remove trailing comma
	stmt += " ON CONFLICT DO NOTHING"

	_, err := db.SQL.Exec(stmt, params...)
	return err
}

// Close implements database.Database
func (db *Database) Close() {
	err := db.SQL.Close()
	if err != nil {
		db.Logger.Error("error while closing connection", "err", err)
	}
}

// -------------------------------------------------------------------------------------------------------------------

// GetLastPruned implements database.PruningDb
func (db *Database) GetLastPruned() (int64, error) {
	var lastPrunedHeight int64
	err := db.SQL.QueryRow(`SELECT coalesce(MAX(last_pruned_height),0) FROM pruning LIMIT 1;`).Scan(&lastPrunedHeight)
	return lastPrunedHeight, err
}

// StoreLastPruned implements database.PruningDb
func (db *Database) StoreLastPruned(height int64) error {
	_, err := db.SQL.Exec(`DELETE FROM pruning`)
	if err != nil {
		return err
	}

	_, err = db.SQL.Exec(`INSERT INTO pruning (last_pruned_height) VALUES ($1)`, height)
	return err
}

// Prune implements database.PruningDb
func (db *Database) Prune(height int64) error {
	_, err := db.SQL.Exec(`DELETE FROM pre_commit WHERE height = $1`, height)
	if err != nil {
		return err
	}

	_, err = db.SQL.Exec(`
DELETE FROM message 
USING transaction 
WHERE message.transaction_hash = transaction.hash AND transaction.height = $1
`, height)
	return err
}
