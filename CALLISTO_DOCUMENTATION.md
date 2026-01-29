# Callisto - Comprehensive Technical Documentation

## Overview

Callisto (formerly BDJuno) is a blockchain indexer built on top of [Juno](https://github.com/forbole/juno) framework, designed specifically for Cosmos-based blockchains. It serves as the backend parser and data layer for [Big Dipper](https://github.com/forbole/big-dipper), a blockchain explorer.

### Purpose
- Parse and index blockchain data from Cosmos SDK-based chains
- Store blockchain data in PostgreSQL for efficient querying
- Provide structured data for GraphQL APIs via Hasura
- Support real-time blockchain monitoring and historical data analysis

---

## Tech Stack

### Core Technologies

#### Backend & Runtime
- **Language**: Go 1.21+
- **Framework**: Juno v6.0.1 (custom blockchain indexing framework)
- **Architecture**: Modular, event-driven architecture

#### Blockchain Integration
- **Cosmos SDK**: v0.50.7
- **CometBFT** (Tendermint): v0.38.7
- **Proto/gRPC**: Protocol Buffers for blockchain communication
- **Codec**: Gogoproto v1.5.0 for serialization

#### Database
- **Database**: PostgreSQL
- **Driver**: lib/pq v1.10.9
- **SQL Extension**: jmoiron/sqlx v1.3.5
- **Features**: 
  - Table partitioning for transactions and messages
  - Custom data types (COIN, DEC_COIN)
  - Stored functions for optimized queries

#### API & Data Layer
- **GraphQL**: Hasura integration
- **REST API**: Built-in actions server (port 3000)
- **gRPC Client**: For blockchain node communication

#### Monitoring & Operations
- **Logging**: Zerolog v1.32.0
- **Metrics**: Prometheus client v1.19.0
- **Scheduler**: gocron v1.37.0 for periodic operations
- **Telemetry**: Built-in telemetry module

#### Development Tools
- **Linting**: golangci-lint v1.55.2
- **Testing**: testify v1.9.0, proullon/ramsql (in-memory SQL for tests)
- **Configuration**: YAML (gopkg.in/yaml.v3)
- **CLI**: Cobra v1.8.0

#### Containerization
- **Docker**: Docker support with Dockerfile
- **Docker Compose**: Multi-service orchestration

---

## System Architecture

### High-Level Flow

```
┌─────────────────────┐
│  Cosmos Blockchain  │
│   (RPC/gRPC/API)   │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│   Callisto Parser   │
│  ┌───────────────┐  │
│  │ Block Module  │  │
│  │ TX Module     │  │
│  │ Msg Module    │  │
│  └───────────────┘  │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│   PostgreSQL DB     │
│  - Blocks           │
│  - Transactions     │
│  - Messages         │
│  - Validators       │
│  - Governance       │
│  - Staking          │
│  - etc.             │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│   Hasura GraphQL    │
│   API Gateway       │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│   Big Dipper UI     │
└─────────────────────┘
```

### Data Flow

1. **Block Reception**: Listen to new blocks from blockchain node
2. **Parsing**: Extract block, transaction, message data
3. **Module Processing**: Each module processes relevant data
4. **Database Storage**: Store parsed data in PostgreSQL
5. **API Exposure**: Hasura exposes data via GraphQL
6. **UI Consumption**: Big Dipper UI queries and displays data

---

## Modules

Callisto implements a modular architecture where each module handles specific blockchain functionality.

### Core Modules (from Juno)

#### 1. **Messages Module**
- **Purpose**: Parse and store transaction messages
- **Responsibilities**:
  - Extract addresses involved in messages
  - Store message type, value, and metadata
  - Support filtering by address and type

#### 2. **Telemetry Module**
- **Purpose**: Monitor system performance
- **Port**: 5000 (configurable)
- **Metrics**: Block processing time, errors, throughput

#### 3. **Pruning Module**
- **Purpose**: Clean old data to manage database size
- **Features**: Configurable retention policies

### Cosmos-Specific Modules

#### 4. **Actions Module** (`modules/actions`)
- **Purpose**: Handle real-time actions and queries
- **Configuration**:
  - Host: 127.0.0.1 (default)
  - Port: 3000
- **Features**: REST API for Hasura actions

#### 5. **Auth Module** (`modules/auth`)
- **Purpose**: Handle account and vesting account data
- **On Every Block**:
  - Parse vesting accounts
  - Store vesting periods
  - Track account creation
- **Database Tables**:
  - `account`: All blockchain accounts
  - `vesting_account`: Vesting account details
  - `vesting_period`: Vesting schedules

#### 6. **Bank Module** (`modules/bank`)
- **Purpose**: Track token supply and balances
- **Periodic Operations** (every 10 mins):
  - Update total supply
- **Database Tables**:
  - `supply`: Total token supply by denom

#### 7. **Consensus Module** (`modules/consensus`)
- **Purpose**: Track consensus state and block metadata
- **On Every Block**:
  - Update consensus state
  - Store block proposer information
  - Track validator pre-commits
- **Database Tables**:
  - `validator`: Validator consensus info
  - `pre_commit`: Validator votes per block
  - `block`: Block metadata

#### 8. **Daily Refetch Module** (`modules/daily_refetch`)
- **Purpose**: Re-fetch missed or stale data
- **Features**: Scheduled data reconciliation

#### 9. **Distribution Module** (`modules/distribution`)
- **Purpose**: Track rewards and community pool
- **On Every Block**:
  - Update community pool
- **Periodic Operations** (every hour):
  - Recalculate distribution metrics
- **Database Tables**:
  - `community_pool`: Community fund balance
  - `distribution_params`: Module parameters

#### 10. **Fee Grant Module** (`modules/feegrant`)
- **Purpose**: Track fee allowances
- **On Every Block**:
  - Parse fee grant messages
  - Store allowance details
- **Database Tables**:
  - (Schema in `11-feegrant.sql`)

#### 11. **Gov Module** (`modules/gov`)
- **Purpose**: Handle governance proposals, votes, and deposits
- **On Every Block**:
  - Parse proposals
  - Track deposits
  - Record votes
  - Calculate tally results
  - Snapshot validator status for proposals
- **Database Tables**:
  - `gov_params`: Governance parameters
  - `proposal`: Proposal details
  - `proposal_deposit`: Deposit records
  - `proposal_vote`: Vote records
  - `proposal_tally_result`: Voting results
  - `proposal_staking_pool_snapshot`: Pool state at proposal
  - `proposal_validator_status_snapshot`: Validator state at proposal

#### 12. **Mint Module** (`modules/mint`)
- **Purpose**: Track inflation and token minting
- **On Every Block**:
  - Update inflation rate
- **Periodic Operations** (every day):
  - Calculate inflation metrics
- **Database Tables**:
  - `mint_params`: Mint module parameters
  - `inflation`: Current inflation rate

#### 13. **Message Type Module** (`modules/message_type`)
- **Purpose**: Catalog all message types in the blockchain
- **On Every Block**:
  - Register new message types
  - Map message types to modules
- **Database Tables**:
  - `message_type`: Message type registry

#### 14. **Modules Module** (`modules/modules`)
- **Purpose**: Track enabled blockchain modules
- **Database Tables**:
  - Module metadata and versions

#### 15. **Price Feed Module** (`modules/pricefeed`)
- **Purpose**: Track token prices and market data
- **Periodic Operations** (every 2 minutes):
  - Fetch token prices
  - Update market cap
- **Database Tables**:
  - (Schema in `07-pricefeed.sql`)

#### 16. **Slashing Module** (`modules/slashing`)
- **Purpose**: Track validator slashing and signing info
- **On Every Block**:
  - Update validator signing info
  - Track missed blocks
  - Record slashing events
- **Database Tables**:
  - `validator_signing_info`: Validator uptime
  - `slashing_params`: Slashing parameters

#### 17. **Staking Module** (`modules/staking`)
- **Purpose**: Track validators, delegations, and staking pool
- **On Every Block**:
  - Update validator information
  - Calculate voting power
  - Track validator status changes
  - Record double sign evidence
- **Periodic Operations**:
  - Calculate staking metrics (hourly/daily)
  - Update delegation ratios
  - Analyze voting power distribution
- **Database Tables**:
  - `staking_params`: Staking parameters
  - `staking_pool`: Total bonded/unbonded tokens
  - `validator_info`: Validator metadata
  - `validator_description`: Validator profile
  - `validator_commission`: Commission rates
  - `validator_voting_power`: Voting power history
  - `validator_status`: Validator state (bonded/unbonded/jailed)
  - `double_sign_vote`: Double sign evidence
  - `double_sign_evidence`: Slashing evidence records

#### 18. **Upgrade Module** (`modules/upgrade`)
- **Purpose**: Track chain upgrades
- **Features**: Monitor and record upgrade proposals

---

## Database Schema

### Schema Organization

The database is organized into multiple schema files, each corresponding to a module:

1. **00-cosmos.sql**: Core blockchain data structures
2. **01-auth.sql**: Accounts and vesting
3. **02-bank.sql**: Supply and balances
4. **03-staking.sql**: Validators and staking
5. **04-consensus.sql**: Consensus state
6. **05-mint.sql**: Inflation and minting
7. **06-distribution.sql**: Rewards and community pool
8. **07-pricefeed.sql**: Token prices
9. **08-gov.sql**: Governance
10. **09-modules.sql**: Module tracking
11. **10-slashing.sql**: Slashing info
12. **11-feegrant.sql**: Fee allowances
13. **12-upgrade.sql**: Chain upgrades

### Core Tables

#### Blockchain Core

**validator**
- `consensus_address` (PK): Validator's consensus address
- `consensus_pubkey`: Public key

**block**
- `height` (PK): Block height
- `hash`: Block hash
- `num_txs`: Number of transactions
- `total_gas`: Gas used
- `proposer_address`: Block proposer
- `timestamp`: Block time

**transaction** (Partitioned by `partition_id`)
- `hash`: Transaction hash
- `height`: Block height
- `success`: Transaction success status
- `messages`: JSON array of messages
- `memo`: Transaction memo
- `signatures`: Signer signatures
- `signer_infos`: JSONB signer information
- `fee`: JSONB fee information
- `gas_wanted`, `gas_used`: Gas metrics
- `raw_log`, `logs`: Transaction logs
- `partition_id`: Partition identifier

**message** (Partitioned by `partition_id`)
- `transaction_hash`: Reference to transaction
- `index`: Message index in transaction
- `type`: Message type
- `value`: JSON message content
- `involved_accounts_addresses`: Array of involved addresses
- `partition_id`: Partition identifier
- `height`: Block height

**message_type**
- `type`: Message type string
- `module`: Module name
- `label`: Display label
- `height`: First seen height

#### Account Management

**account**
- `address` (PK): Account address

**vesting_account**
- `type`: Vesting account type
- `address`: Account address
- `original_vesting`: Original vesting coins
- `end_time`: Vesting end time
- `start_time`: Vesting start time (optional)

**vesting_period**
- `vesting_account_id`: Reference to vesting account
- `period_order`: Period sequence number
- `length`: Period length
- `amount`: Coins vesting in this period

#### Staking

**validator_info**
- `consensus_address` (PK): Validator consensus address
- `operator_address`: Operator address
- `self_delegate_address`: Self-delegation address
- `max_change_rate`, `max_rate`: Commission rate limits

**validator_description**
- `validator_address`: Validator address
- `moniker`: Validator name
- `identity`: Keybase identity
- `avatar_url`: Avatar URL
- `website`, `security_contact`, `details`: Metadata

**validator_commission**
- `validator_address`: Validator address
- `commission`: Current commission rate
- `min_self_delegation`: Minimum self-delegation

**validator_voting_power**
- `validator_address`: Validator address
- `voting_power`: Current voting power
- `height`: Block height

**validator_status**
- `validator_address`: Validator address
- `status`: Status (bonded/unbonded/unbonding)
- `jailed`: Jailed status
- `height`: Block height

**staking_pool**
- `bonded_tokens`: Total bonded tokens
- `not_bonded_tokens`: Total not bonded tokens
- `unbonding_tokens`: Total unbonding tokens
- `staked_not_bonded_tokens`: Staked but not bonded
- `height`: Block height

#### Governance

**proposal**
- `id` (PK): Proposal ID
- `title`, `description`: Proposal details
- `metadata`: Additional metadata
- `content`: JSONB proposal content
- `submit_time`: Submission time
- `deposit_end_time`, `voting_start_time`, `voting_end_time`: Proposal timeline
- `proposer_address`: Proposer
- `status`: Proposal status

**proposal_deposit**
- `proposal_id`: Proposal reference
- `depositor_address`: Depositor
- `amount`: Deposit amount
- `timestamp`: Deposit time
- `transaction_hash`: Transaction reference

**proposal_vote**
- `proposal_id`: Proposal reference
- `voter_address`: Voter
- `option`: Vote option
- `weight`: Vote weight
- `timestamp`: Vote time

**proposal_tally_result**
- `proposal_id`: Proposal reference
- `yes`, `abstain`, `no`, `no_with_veto`: Vote counts

#### Other Modules

**supply**
- `coins`: Total supply by denomination
- `height`: Block height

**community_pool**
- `coins`: Community pool balance
- `height`: Block height

**inflation**
- `value`: Current inflation rate
- `height`: Block height

**validator_signing_info**
- `validator_address`: Validator address
- `start_height`: Start height
- `index_offset`: Index offset
- `jailed_until`: Jail expiration
- `tombstoned`: Tombstone status
- `missed_blocks_counter`: Missed blocks count

### Custom Data Types

**COIN**
```sql
CREATE TYPE COIN AS (
    denom  TEXT,
    amount TEXT
);
```

**DEC_COIN** (Decimal Coin for precision)
```sql
CREATE TYPE DEC_COIN AS (
    denom  TEXT,
    amount TEXT
);
```

### Stored Functions

**messages_by_address**
- Filter messages by involved addresses and types
- Pagination support
- Returns message records

**messages_by_type**
- Filter messages by type
- Pagination support
- Returns message records

### Database Features

#### Partitioning
- **transaction** and **message** tables are partitioned by `partition_id`
- Improves query performance for large datasets
- Configurable partition size and batch size

#### Indexing
- Comprehensive indexes on frequently queried columns
- GIN indexes for array fields (e.g., `involved_accounts_addresses`)
- Height-based indexes for time-series queries

#### Autovacuum Tuning
- Block table optimized for high-insert workload
- Custom autovacuum thresholds

---

## Configuration

### Config File Structure (`config.yaml`)

```yaml
chain:
  bech32_prefix: dydx           # Chain address prefix
  modules: []                   # Enabled modules

node:
  type: remote                  # Node type (local/remote)
  config:
    rpc:
      client_name: perpx
      address: https://rpc-perpx-testnet.1119labs.com
    grpc:
      address: grpc://grpc-perpx-testnet.1119labs.com
      insecure: true
    api:
      address: https://api-perpx-testnet.1119labs.com

parsing:
  workers: 5                    # Number of concurrent workers
  start_height: 1               # Starting block height
  average_block_time: 2s        # Expected block time
  listen_new_blocks: true       # Listen for new blocks
  parse_old_blocks: true        # Parse historical blocks
  parse_genesis: true           # Parse genesis state
  fast_sync: false              # Fast sync mode

database:
  url: postgresql://user:pass@host:port/db
  max_open_connections: 1
  max_idle_connections: 1
  partition_size: 100000        # Rows per partition
  partition_batch: 1000         # Batch size for partitioning
  ssl_mode_enable: "false"
  ssl_root_cert: ""
  ssl_cert: ""
  ssl_key: ""

logging:
  level: debug                  # Log level (debug/info/warn/error)
  format: text                  # Log format (text/json)

actions:
  host: 127.0.0.1               # Actions server host
  port: 3000                    # Actions server port

telemetry:
  enabled: false                # Enable telemetry
  port: 5000                    # Metrics port
```

---

## Commands & Operations

### Main Commands

#### 1. **init**
Initialize Callisto configuration
```bash
callisto init
```

#### 2. **parse**
Parse specific blockchain data
```bash
callisto parse <subcommand>
```

Subcommands:
- `genesis`: Parse genesis file
- `blocks`: Parse specific block range
- `transactions`: Parse specific transactions

#### 3. **migrate**
Run database migrations
```bash
callisto migrate
```

#### 4. **start**
Start the indexer
```bash
callisto start
```

Features:
- Continuous block parsing
- Real-time indexing
- Periodic operations execution

#### 5. **version**
Display version information
```bash
callisto version
```

---

## RPC/gRPC Data Fetching Mechanism

### Connection Architecture

Callisto uses a **dual-protocol approach** for blockchain communication:

1. **RPC (Tendermint/CometBFT)**: For block and transaction data
2. **gRPC (Cosmos SDK)**: For module-specific queries

### Connection Setup

#### 1. **Node Configuration**

```yaml
node:
  type: remote  # or 'local'
  config:
    rpc:
      client_name: perpx
      address: https://rpc-perpx-testnet.1119labs.com
      max_connections: 20  # HTTP connection pool
    grpc:
      address: grpc://grpc-perpx-testnet.1119labs.com
      insecure: true  # TLS settings
    api:
      address: https://api-perpx-testnet.1119labs.com
```

#### 2. **RPC Client Initialization**

```go
// Creates HTTP client with custom transport
httpClient := jsonrpcclient.DefaultHTTPClient(rpcAddress)
httpTransport.MaxConnsPerHost = maxConnections

// Creates Tendermint RPC client
rpcClient := httpclient.NewWithClient(rpcAddress, "/websocket", httpClient)
rpcClient.Start()
```

**Key Features**:
- WebSocket support for event subscriptions
- Connection pooling (configurable)
- Automatic reconnection
- Context-based request management

#### 3. **gRPC Client Initialization**

```go
// Create gRPC connection
grpcConn := grpc.Dial(
    grpcAddress,
    grpc.WithTransportCredentials(credentials),
)

// Initialize module-specific clients
bankClient := banktypes.NewQueryClient(grpcConn)
stakingClient := stakingtypes.NewQueryClient(grpcConn)
govClient := govtypes.NewQueryClient(grpcConn)
// ... more clients
```

**Transport Options**:
- TLS/SSL support (configurable)
- Insecure mode for development
- HTTP/2 based communication
- Multiplexed streams

### Data Source Abstraction

Callisto implements a **Source Pattern** that abstracts data fetching from either remote nodes or local nodes.

#### Source Interface Hierarchy

```
Source (Interface)
├── Remote Source (gRPC/RPC)
│   ├── Bank Source
│   ├── Staking Source
│   ├── Gov Source
│   ├── Distribution Source
│   ├── Mint Source
│   └── Slashing Source
└── Local Source (Direct Keeper Access)
    └── (Same module sources)
```

#### Building Sources

```go
func BuildSources(nodeConfig) (*Sources, error) {
    // Remote node: Uses gRPC
    if nodeType == remote {
        grpcSource := remote.NewSource(grpcConfig)
        
        return &Sources{
            BankSource: NewBankSource(
                grpcSource, 
                banktypes.NewQueryClient(grpcConn)
            ),
            StakingSource: NewStakingSource(
                grpcSource,
                stakingtypes.NewQueryClient(grpcConn)
            ),
            // ... more sources
        }
    }
    
    // Local node: Direct keeper access
    if nodeType == local {
        localSource := local.NewSource(homeDir, codec)
        
        return &Sources{
            BankSource: NewLocalBankSource(
                localSource,
                app.BankKeeper
            ),
            // ... more sources
        }
    }
}
```

### RPC Data Fetching (Block-level)

#### Available RPC Methods

```go
// Node interface - Tendermint RPC calls
type Node interface {
    // Core block data
    Block(height int64) (*ResultBlock, error)
    BlockResults(height int64) (*ResultBlockResults, error)
    
    // Validators
    Validators(height int64) (*ResultValidators, error)
    
    // Chain info
    LatestHeight() (int64, error)
    ChainID() (string, error)
    Genesis() (*ResultGenesis, error)
    ConsensusState() (*RoundStateSimple, error)
    
    // Transactions
    Tx(hash string) (*Transaction, error)
    TxSearch(query, orderBy string, page, perPage int) (*ResultTxSearch, error)
    
    // Event subscription
    SubscribeNewBlocks(subscriber string) (<-chan ResultEvent, error)
    SubscribeEvents(subscriber, query string) (<-chan ResultEvent, error)
}
```

#### Block Fetching Process

```go
// 1. Subscribe to new blocks (real-time mode)
eventCh, cancel, err := node.SubscribeNewBlocks("callisto")

// 2. Receive block event
for event := range eventCh {
    blockHeight := event.Data.(EventDataNewBlock).Block.Height
    
    // 3. Fetch full block data
    block, err := node.Block(blockHeight)
    // Returns: block metadata, proposer, transactions, timestamp
    
    // 4. Fetch block execution results
    blockResults, err := node.BlockResults(blockHeight)
    // Returns: begin_block events, end_block events, validator updates
    
    // 5. Fetch validators at this height
    validators, err := node.Validators(blockHeight)
    // Returns: validator set with voting power
}
```

#### Transaction Fetching

```go
// Method 1: From block
block, _ := node.Block(height)
for _, txBytes := range block.Block.Txs {
    // Decode transaction from bytes
}

// Method 2: By hash (using REST API)
tx, err := node.Tx("ABC123...") 
// Makes HTTP call to: /cosmos/tx/v1beta1/txs/{hash}

// Method 3: Search transactions
results, err := node.TxSearch(
    "tx.height >= 100 AND message.action='/cosmos.bank.v1beta1.MsgSend'",
    "asc",
    &page,
    &perPage,
)
```

### gRPC Data Fetching (Module-level)

#### Height-Aware Context

gRPC queries support **historical state queries** using height context:

```go
// Create context with height header
func GetHeightRequestContext(ctx context.Context, height int64) context.Context {
    return metadata.AppendToOutgoingContext(
        ctx,
        grpctypes.GRPCBlockHeightHeader,  // "x-cosmos-block-height"
        strconv.FormatInt(height, 10),
    )
}

// Use in queries
ctx := GetHeightRequestContext(context.Background(), 12345)
response, err := stakingClient.Validator(ctx, &QueryValidatorRequest{...})
```

This allows querying **historical state** at any block height.

#### Module-Specific Queries

**1. Staking Module**

```go
type StakingSource interface {
    // Single validator
    GetValidator(height int64, operatorAddr string) (Validator, error)
    
    // All validators with status
    GetValidatorsWithStatus(height int64, status string) ([]Validator, error)
    
    // Staking pool
    GetPool(height int64) (Pool, error)
    
    // Staking parameters
    GetParams(height int64) (Params, error)
    
    // Delegations (paginated)
    GetDelegationsWithPagination(
        height int64,
        delegator string,
        pagination *PageRequest,
    ) (*QueryDelegatorDelegationsResponse, error)
}
```

**Implementation**:
```go
func (s *RemoteSource) GetValidator(height int64, valOper string) (Validator, error) {
    ctx := GetHeightRequestContext(s.Ctx, height)
    
    res, err := s.stakingClient.Validator(ctx, &QueryValidatorRequest{
        ValidatorAddr: valOper,
    })
    
    return res.Validator, err
}
```

**2. Bank Module**

```go
type BankSource interface {
    // Account balances
    GetBalances(addresses []string, height int64) ([]AccountBalance, error)
    
    // Total supply
    GetSupply(height int64) (Coins, error)
    
    // Single account balance
    GetAccountBalance(address string, height int64) ([]Coin, error)
}
```

**Implementation with Pagination**:
```go
func (s *RemoteSource) GetSupply(height int64) (Coins, error) {
    ctx := GetHeightRequestContext(s.Ctx, height)
    
    var coins []Coin
    var nextKey []byte
    
    // Paginated fetching
    for {
        res, err := s.bankClient.TotalSupply(ctx, &QueryTotalSupplyRequest{
            Pagination: &PageRequest{
                Key:   nextKey,
                Limit: 100,  // Fetch 100 at a time
            },
        })
        
        coins = append(coins, res.Supply...)
        
        if len(res.Pagination.NextKey) == 0 {
            break  // No more pages
        }
        nextKey = res.Pagination.NextKey
    }
    
    return coins, nil
}
```

**3. Gov Module**

```go
type GovSource interface {
    // Proposals
    Proposal(height int64, id uint64) (Proposal, error)
    ProposalDeposit(height int64, id uint64, depositor string) (*Deposit, error)
    ProposalVote(height int64, id uint64, voter string) (*Vote, error)
    TallyResult(height int64, id uint64) (*TallyResult, error)
    
    // Parameters
    DepositParams(height int64) (DepositParams, error)
    VotingParams(height int64) (VotingParams, error)
    TallyParams(height int64) (TallyParams, error)
}
```

**4. Distribution Module**

```go
type DistrSource interface {
    // Community pool
    CommunityPool(height int64) (DecCoins, error)
    
    // Validator commission
    ValidatorCommission(height int64, valAddr string) (DecCoins, error)
    
    // Delegation rewards
    DelegationRewards(height int64, delegator, validator string) (DecCoins, error)
    
    // Total delegation rewards
    DelegationTotalRewards(height int64, delegator string) (*QueryDelegationTotalRewardsResponse, error)
}
```

**5. Mint Module**

```go
type MintSource interface {
    // Current inflation
    Inflation(height int64) (Dec, error)
    
    // Mint parameters
    Params(height int64) (Params, error)
}
```

**6. Slashing Module**

```go
type SlashingSource interface {
    // Signing info
    SigningInfo(height int64, consAddr string) (ValidatorSigningInfo, error)
    
    // All signing infos (paginated)
    SigningInfos(height int64, pagination *PageRequest) ([]ValidatorSigningInfo, error)
    
    // Slashing parameters
    Params(height int64) (Params, error)
}
```

### Pagination Strategy

For endpoints returning large datasets, Callisto uses **automatic pagination**:

```go
func FetchAllWithPagination[T any](
    fetchFunc func(nextKey []byte) (items []T, nextKey []byte, err error),
) ([]T, error) {
    var allItems []T
    var nextKey []byte
    
    for {
        items, key, err := fetchFunc(nextKey)
        if err != nil {
            return nil, err
        }
        
        allItems = append(allItems, items...)
        
        if len(key) == 0 {
            break  // Last page
        }
        nextKey = key
    }
    
    return allItems, nil
}
```

**Example: Fetch all validators**
```go
func (s *Source) GetValidatorsWithStatus(height int64, status string) ([]Validator, error) {
    ctx := GetHeightRequestContext(s.Ctx, height)
    
    var validators []Validator
    var nextKey []byte
    
    for {
        res, err := s.stakingClient.Validators(ctx, &QueryValidatorsRequest{
            Status: status,
            Pagination: &PageRequest{
                Key:   nextKey,
                Limit: 100,  // Query 100 validators at a time
            },
        })
        
        validators = append(validators, res.Validators...)
        
        if len(res.Pagination.NextKey) == 0 {
            break
        }
        nextKey = res.Pagination.NextKey
    }
    
    return validators, nil
}
```

### Error Handling & Retries

```go
// Typical error handling pattern
func (s *Source) GetValidator(height int64, addr string) (Validator, error) {
    ctx := GetHeightRequestContext(s.Ctx, height)
    
    res, err := s.stakingClient.Validator(ctx, &QueryValidatorRequest{
        ValidatorAddr: addr,
    })
    
    if err != nil {
        return Validator{}, fmt.Errorf("error while getting validator: %w", err)
    }
    
    return res.Validator, nil
}
```

**Common Error Scenarios**:
- gRPC connection failures → Logged and retried
- Invalid height (pruned) → Skip or use latest
- Timeout → Configurable timeout context
- Rate limiting → Backoff strategy

### Genesis Data Fetching

Genesis data is fetched specially:

```go
func (node *Node) Genesis() (*ResultGenesis, error) {
    // Try standard genesis endpoint
    res, err := node.client.Genesis(ctx)
    
    // If too large, use chunked API
    if err != nil && strings.Contains(err.Error(), "genesis_chunked") {
        return node.getGenesisChunked()
    }
    
    return res, err
}

func (node *Node) getGenesisChunked() (*ResultGenesis, error) {
    // Fetch chunks sequentially
    var genesisData []byte
    
    for chunkID := 0; ; chunkID++ {
        chunk, err := node.client.GenesisChunked(ctx, chunkID)
        
        decoded, _ := base64.StdEncoding.DecodeString(chunk.Data)
        genesisData = append(genesisData, decoded...)
        
        if chunkID == chunk.TotalChunks-1 {
            break
        }
    }
    
    // Parse complete genesis
    var genDoc GenesisDoc
    json.Unmarshal(genesisData, &genDoc)
    
    return &ResultGenesis{Genesis: &genDoc}, nil
}
```

### Event Subscription (Real-time)

For **live indexing**, Callisto subscribes to blockchain events via WebSocket:

```go
// Subscribe to new blocks
eventCh, cancel, err := node.SubscribeNewBlocks("callisto-subscriber")
defer cancel()

// Listen for events
for event := range eventCh {
    switch event.Type {
    case "NewBlock":
        block := event.Data.(EventDataNewBlock)
        // Process block immediately
        
    case "Tx":
        tx := event.Data.(EventDataTx)
        // Process transaction
    }
}
```

**Subscription Queries**:
```go
// New blocks
"tm.event = 'NewBlock'"

// Specific message types
"message.action = '/cosmos.bank.v1beta1.MsgSend'"

// Custom events
"transfer.recipient = 'cosmos1...'"
```

### Data Flow Summary

```
┌─────────────────────────────────────────────────────────┐
│                    Blockchain Node                      │
│  ┌──────────────┐              ┌──────────────┐        │
│  │ Tendermint   │              │  Cosmos SDK  │        │
│  │ RPC Server   │              │ gRPC Server  │        │
│  │ :26657       │              │ :9090        │        │
│  └──────┬───────┘              └──────┬───────┘        │
└─────────┼──────────────────────────────┼───────────────┘
          │                               │
          │ HTTP/WebSocket                │ gRPC (HTTP/2)
          │                               │
┌─────────▼───────────────────────────────▼───────────────┐
│              Callisto Indexer                           │
│  ┌──────────────────┐      ┌──────────────────┐        │
│  │   RPC Client     │      │   gRPC Clients   │        │
│  │ (Block/TX/Event) │      │  (Module Queries)│        │
│  └────────┬─────────┘      └────────┬─────────┘        │
│           │                         │                   │
│           │    ┌────────────────────┤                   │
│           │    │                    │                   │
│           ▼    ▼                    ▼                   │
│  ┌────────────────┐      ┌──────────────────┐          │
│  │  Block Parser  │      │  Module Sources  │          │
│  └────────┬───────┘      └────────┬─────────┘          │
│           │                       │                     │
│           └───────────┬───────────┘                     │
│                       ▼                                 │
│           ┌──────────────────────┐                      │
│           │  Database Writer     │                      │
│           └──────────┬───────────┘                      │
└──────────────────────┼──────────────────────────────────┘
                       │
                       ▼
              ┌─────────────────┐
              │   PostgreSQL    │
              └─────────────────┘
```

### Performance Optimizations

1. **Connection Pooling**: Reuse HTTP/gRPC connections
2. **Parallel Workers**: Multiple goroutines process blocks concurrently
3. **Batch Queries**: Group similar queries when possible
4. **Pagination**: Fetch large datasets in chunks (default: 100 items)
5. **Caching**: Cache frequently accessed data (params, validators)
6. **Context Timeouts**: Prevent hanging requests
7. **Retry Logic**: Automatic retry on transient failures

### Configuration Best Practices

```yaml
node:
  config:
    rpc:
      max_connections: 20  # Tune based on load
    grpc:
      insecure: false  # Use TLS in production

parsing:
  workers: 5  # Number of concurrent block processors
  average_block_time: 2s  # Optimize polling interval

database:
  max_open_connections: 10  # Balance with worker count
```

---

## Processing Flow

### Block Processing Flow

```
1. New Block Detected
   ↓
2. Fetch Block Data (RPC)
   - node.Block(height) → block metadata, txs
   - node.BlockResults(height) → events, results
   - node.Validators(height) → validator set
   ↓
3. Parse Block Metadata
   - Extract proposer
   - Parse pre-commits
   - Store block info
   ↓
4. Store Block in Database
   ↓
5. Parse Transactions
   - For each TX in block
   - Fetch full TX details via REST API
   ↓
6. For Each Transaction:
   - Parse Messages
   - Extract Involved Addresses
   - Store Transaction
   - Store Messages
   ↓
7. Module Processing (via gRPC):
   - Auth Module: Parse vesting accounts
   - Staking Module: 
     * GetValidators() → Update validator info
     * GetPool() → Update staking pool
   - Gov Module: 
     * GetProposal() → Update proposals
     * GetVotes() → Store votes
   - Distribution Module:
     * GetCommunityPool() → Update pool balance
   - Mint Module:
     * GetInflation() → Update inflation
   - Slashing Module:
     * GetSigningInfos() → Update signing info
   - Bank Module:
     * GetSupply() → Update total supply
   ↓
8. Commit to Database
   ↓
9. Update Prometheus Metrics
   ↓
10. Wait for Next Block
```

### Worker Queue System

Callisto uses a **buffered channel-based queue** for block height processing:

```go
// HeightQueue is a buffered channel
type HeightQueue chan int64

// Create queue with buffer size
exportQueue := NewQueue(25)  // Buffer of 25 heights
```

**Queue Flow**:
```
Producer (Enqueuer)           Queue (Channel)           Consumer (Worker)
     │                             │                          │
     ├─ enqueue height ──────────► │                          │
     ├─ enqueue height ──────────► │                          │
     │                             ├───────► height ─────────►├─ Process
     │                             │                          │
     ├─ enqueue height ──────────► │                          │
     │                             ├───────► height ─────────►├─ Process
                                   │                          │
                              [Buffer: 25]              [5 Workers]
```

### New Block Fetching Mechanism

**Real-time Block Indexing**

```go
func enqueueNewBlocks(exportQueue HeightQueue, ctx *Context) {
    currHeight := mustGetLatestHeight(ctx)
    
    // Infinite loop for continuous monitoring
    for {
        latestBlockHeight := mustGetLatestHeight(ctx)
        
        // Enqueue all new heights
        for ; currHeight <= latestBlockHeight; currHeight++ {
            logger.Debug("enqueueing new block", "height", currHeight)
            exportQueue <- currHeight  // Send to queue
        }
        
        // Wait for next block (average block time)
        time.Sleep(config.GetAvgBlockTime())  // e.g., 2 seconds
    }
}
```

**Step-by-Step Process**:

1. **Get Current Height**
   ```go
   currHeight := mustGetLatestHeight(ctx)
   // Returns: 12345 (example)
   ```

2. **Polling Loop** (runs forever)
   - Sleep for `average_block_time` (configured, e.g., 2s)
   - Query RPC for latest height
   ```go
   latestBlockHeight := node.LatestHeight()
   // Returns: 12347 (2 blocks ahead)
   ```

3. **Enqueue New Heights**
   - If new blocks detected (12346, 12347)
   - Send each height to queue
   ```go
   exportQueue <- 12346
   exportQueue <- 12347
   ```

4. **Workers Process** (see Worker section below)

**Retry Logic for RPC Failures**:
```go
func mustGetLatestHeight(ctx *Context) int64 {
    for retryCount := 0; retryCount < 50; retryCount++ {
        latestBlockHeight, err := ctx.Node.LatestHeight()
        if err == nil {
            return latestBlockHeight
        }
        
        logger.Error("failed to get latest height",
            "retry_count", retryCount,
            "retry_interval", avgBlockTime)
        
        time.Sleep(avgBlockTime)  // Wait before retry
    }
    
    return 0  // Failed after 50 retries
}
```

### Missing Block Fetching (Historical Sync)

**Catch-up Mechanism for Old Blocks**

```go
func enqueueMissingBlocks(exportQueue HeightQueue, ctx *Context) {
    cfg := config.Cfg.Parser
    
    // 1. Get latest blockchain height
    latestBlockHeight := mustGetLatestHeight(ctx)
    // Example: 100000
    
    // 2. Get last indexed height in database
    lastDbBlockHeight, err := ctx.Database.GetLastBlockHeight()
    // Example: 95000
    
    // 3. Determine start height
    startHeight := cfg.StartHeight  // From config, e.g., 1
    if startHeight == 0 {
        startHeight = max(1, lastDbBlockHeight)
    }
    
    // 4. Fast sync option
    if cfg.FastSync {
        logger.Info("fast sync enabled, downloading state snapshot")
        
        for _, module := range ctx.Modules {
            if fastSyncModule, ok := module.(FastSyncModule); ok {
                fastSyncModule.DownloadState(latestBlockHeight)
            }
        }
        return
    }
    
    // 5. Normal sync: Get missing heights
    missingHeights := ctx.Database.GetMissingHeights(startHeight, latestBlockHeight)
    // SQL: SELECT generate_series($1, $2) EXCEPT SELECT height FROM block
    // Returns: [95001, 95002, ..., 99999, 100000]
    
    // 6. Enqueue all missing blocks
    for _, height := range missingHeights {
        logger.Debug("enqueueing missing block", "height", height)
        exportQueue <- height
    }
}
```

**Database Query for Missing Heights**:
```sql
-- Find gaps in block sequence
SELECT generate_series($1::int, $2::int) 
EXCEPT 
SELECT height FROM block 
ORDER BY 1;

-- Example result: [100, 105, 200, 201]
-- (blocks 100, 105, 200, 201 are missing)
```

### Daily Refetch Mechanism

**Automatic Re-indexing of Recent Blocks**

The Daily Refetch module ensures data consistency by re-checking recent blocks.

```go
func (m *Module) RegisterPeriodicOperations(scheduler *Scheduler) error {
    // Run every day at midnight (00:00)
    scheduler.Every(1).Day().At("00:00").Do(func() {
        m.refetchMissingBlocks()
    })
}
```

**Refetch Process**:

```go
func refetchMissingBlocks() error {
    // 1. Get current latest height
    latestBlock := node.LatestHeight()
    // Example: 100000
    
    // 2. Get height from 24 hours ago
    blockHeightDayAgo := database.GetBlockHeightTimeDayAgo(now())
    // Example: 85000 (assuming 6s block time = 14400 blocks/day)
    
    startHeight := blockHeightDayAgo.Height
    
    // 3. Find missing blocks in range
    missingBlocks := database.GetMissingBlocks(startHeight, latestBlock)
    // SQL: SELECT generate_series(85000, 100000) 
    //      EXCEPT SELECT height FROM block
    // Returns: [85123, 85456, 99999] (example gaps)
    
    // 4. Re-fetch each missing block
    for _, blockHeight := range missingBlocks {
        err := worker.Process(blockHeight)
        if err != nil {
            return fmt.Errorf("error re-fetching block %d: %w", blockHeight, err)
        }
    }
    
    return nil
}
```

**Why Daily Refetch?**
- **Network Issues**: Blocks missed due to temporary RPC failures
- **Data Consistency**: Verify data integrity
- **Reorg Handling**: Detect and fix chain reorganizations
- **Gap Filling**: Catch any blocks that slipped through

**Schedule**: Runs at 00:00 UTC daily via cron scheduler

### Worker Processing

**Multi-Worker Architecture**

```go
// Create N workers (configurable, default: 5)
workers := make([]Worker, cfg.Workers)

for i := range workers {
    workers[i] = NewWorker(ctx, exportQueue, i)
    go workers[i].Start()  // Start in goroutine
}
```

**Worker Lifecycle**:

```go
func (w Worker) Start() {
    WorkerCount.Inc()  // Prometheus metric
    
    // Consume from queue forever
    for height := range w.queue {
        err := w.ProcessIfNotExists(height)
        
        if err != nil {
            // REQUEUE on failure
            time.Sleep(avgBlockTime)
            
            go func() {
                logger.Error("re-enqueueing failed block", 
                    "height", height, 
                    "err", err)
                w.queue <- height  // Put back in queue
            }()
        }
        
        // Update metrics
        WorkerHeight.
            WithLabelValues(fmt.Sprintf("%d", w.index), chainID).
            Set(float64(height))
    }
}
```

**ProcessIfNotExists Logic**:

```go
func (w Worker) ProcessIfNotExists(height int64) error {
    // 1. Check if block already exists
    exists, err := w.db.HasBlock(height)
    if err != nil {
        return fmt.Errorf("error checking block: %w", err)
    }
    
    if exists {
        logger.Debug("skipping already exported block", "height", height)
        return nil  // Skip processing
    }
    
    // 2. Block doesn't exist, process it
    return w.Process(height)
}
```

**Process Logic** (Core Processing):

```go
func (w Worker) Process(height int64) error {
    // Special case: Genesis
    if height == 0 {
        genesisDoc, genesisState := GetGenesis(node)
        return w.HandleGenesis(genesisDoc, genesisState)
    }
    
    logger.Debug("processing block", "height", height)
    
    // 1. Fetch block data from RPC
    block, err := w.node.Block(height)
    if err != nil {
        return fmt.Errorf("failed to get block: %w", err)
    }
    
    // 2. Fetch block results (events)
    events, err := w.node.BlockResults(height)
    if err != nil {
        return fmt.Errorf("failed to get block results: %w", err)
    }
    
    // 3. Fetch transactions
    txs, err := w.node.Txs(block)
    if err != nil {
        return fmt.Errorf("failed to get transactions: %w", err)
    }
    
    // 4. Fetch validators
    vals, err := w.node.Validators(height)
    if err != nil {
        return fmt.Errorf("failed to get validators: %w", err)
    }
    
    // 5. Export everything
    return w.ExportBlock(block, events, txs, vals)
}
```

**ExportBlock** (Database Operations):

```go
func (w Worker) ExportBlock(block, results, txs, validators) error {
    // 1. Save validators
    err := w.SaveValidators(validators)
    
    // 2. Save block metadata
    err = w.db.SaveBlock(NewBlockFromTmBlock(block, totalGas))
    
    // 3. Save validator commits/signatures
    err = w.ExportCommit(block.LastCommit, validators)
    
    // 4. Call module handlers (per block)
    for _, module := range w.modules {
        if blockModule, ok := module.(BlockModule); ok {
            err = blockModule.HandleBlock(block, results, txs, validators)
            // Examples:
            // - Staking: Update validator info
            // - Gov: Update proposals
            // - Distribution: Update community pool
        }
    }
    
    // 5. Export transactions
    return w.ExportTxs(txs)
}
```

**ExportTxs** (Transaction Processing):

```go
func (w Worker) ExportTxs(txs []*Transaction) error {
    for _, tx := range txs {
        // 1. Save transaction
        err := w.saveTx(tx)
        
        // 2. Call transaction handlers
        w.handleTx(tx)
        
        // 3. Handle each message in transaction
        for i, msg := range tx.Body.Messages {
            w.handleMessage(i, msg, tx)
            // Examples:
            // - MsgSend: Extract sender/receiver
            // - MsgDelegate: Track delegation
            // - MsgVote: Record governance vote
        }
    }
    
    // Update metrics
    totalBlocks := w.db.GetTotalBlocks()
    DbBlockCount.Set(float64(totalBlocks))
    
    return nil
}
```

### Requeue (Error Recovery) Mechanism

**Automatic Retry on Failures**

When a worker fails to process a block, Callisto **automatically requeues** it:

```go
func (w Worker) Start() {
    for height := range w.queue {
        err := w.ProcessIfNotExists(height)
        
        if err != nil {
            // ===== REQUEUE LOGIC =====
            
            // 1. Wait before retry (backoff)
            time.Sleep(avgBlockTime)
            
            // 2. Re-enqueue in separate goroutine (non-blocking)
            go func(h int64) {
                logger.Error("re-enqueueing failed block",
                    "height", h,
                    "error", err,
                    "worker", w.index)
                
                w.queue <- h  // Put back in queue
            }(height)
            
            // Worker continues to next item
        }
    }
}
```

**Failure Scenarios & Handling**:

| Failure Type | Cause | Action |
|--------------|-------|--------|
| RPC Timeout | Node unresponsive | Requeue, retry |
| Network Error | Connection lost | Requeue, retry |
| Parse Error | Invalid data | Log error, requeue |
| DB Write Error | Database issue | Requeue, retry |
| Missing Validator | Data inconsistency | Requeue, retry |

**Retry Strategy**:
- **Immediate Requeue**: Block goes back to end of queue
- **Backoff**: Wait `average_block_time` before requeue
- **Infinite Retries**: No max retry limit (will retry forever)
- **Non-blocking**: Requeue happens in goroutine, worker continues

**TODO: Future Improvements**:
```go
// Currently commented in code:
// TODO: Implement exponential backoff or max retries for a block height.

// Proposed implementation:
type RetryInfo struct {
    Height      int64
    Attempts    int
    LastAttempt time.Time
}

// Exponential backoff: 2s, 4s, 8s, 16s, 32s, 60s (max)
backoffDuration := min(
    avgBlockTime * (1 << attempts),  // 2^attempts
    60 * time.Second,
)
```

### Complete Flow Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                    START COMMAND                                 │
└────────────┬────────────────────────────────────────────────────┘
             │
             ├──► Start Periodic Operations (Scheduler)
             │    └─► Daily Refetch (00:00 daily)
             │
             ├──► Create Queue (buffer: 25)
             │
             ├──► Start N Workers (default: 5)
             │    └─► Each worker: for height := range queue { ... }
             │
             ├──► [Config: parse_genesis = true]
             │    └─► Enqueue height 0
             │
             ├──► [Config: parse_old_blocks = true]
             │    └─► enqueueMissingBlocks() → Historical Sync
             │         1. Get start height (config or last DB height)
             │         2. Get latest height from RPC
             │         3. Query missing heights from DB
             │         4. Enqueue all missing heights
             │
             └──► [Config: listen_new_blocks = true]
                  └─► enqueueNewBlocks() → Real-time Sync
                       Loop forever:
                         1. Get latest height from RPC
                         2. Enqueue all new heights
                         3. Sleep avg_block_time (2s)
                         4. Repeat

┌─────────────────────────────────────────────────────────────────┐
│                    QUEUE (Channel)                               │
│  [Height: 1000] → [Height: 1001] → [Height: 1002] → ...         │
│                   Buffer Size: 25                                │
└────────────┬────────────────────────────────────────────────────┘
             │
             ├──► Worker 0 ──► Process Height
             ├──► Worker 1 ──► Process Height
             ├──► Worker 2 ──► Process Height
             ├──► Worker 3 ──► Process Height
             └──► Worker 4 ──► Process Height

┌─────────────────────────────────────────────────────────────────┐
│                    WORKER PROCESSING                             │
└────────────┬────────────────────────────────────────────────────┘
             │
             ├──► Check if block exists in DB
             │    └─► YES: Skip
             │    └─► NO: Continue
             │
             ├──► Fetch Block Data (RPC)
             │    ├─► Block metadata
             │    ├─► Block results (events)
             │    ├─► Transactions
             │    └─► Validators
             │
             ├──► Export to Database
             │    ├─► Save validators
             │    ├─► Save block
             │    ├─► Save commits
             │    ├─► Call module handlers
             │    └─► Save transactions & messages
             │
             ├──► SUCCESS
             │    └─► Update metrics, continue
             │
             └──► FAILURE
                  └─► REQUEUE MECHANISM
                       1. Sleep avg_block_time
                       2. Log error
                       3. Re-enqueue height to queue
                       4. Worker continues to next height

┌─────────────────────────────────────────────────────────────────┐
│                    DAILY REFETCH (00:00 UTC)                     │
└────────────┬────────────────────────────────────────────────────┘
             │
             ├──► Get latest height
             ├──► Get height from 24h ago
             ├──► Find missing blocks in range
             ├──► Process each missing block directly
             └──► Log results
```

### Configuration Options

```yaml
parsing:
  workers: 5                    # Number of concurrent workers
  start_height: 1               # Starting block height
  average_block_time: 2s        # Expected block time (for polling/backoff)
  listen_new_blocks: true       # Enable real-time new block fetching
  parse_old_blocks: true        # Enable historical block syncing
  parse_genesis: true           # Parse genesis block (height 0)
  fast_sync: false              # Use state snapshot instead of full sync
```

**Processing Modes**:

1. **Full Sync** (Recommended for new deployments)
   ```yaml
   parse_genesis: true
   parse_old_blocks: true
   listen_new_blocks: true
   start_height: 1
   ```
   - Starts from genesis
   - Syncs all historical blocks
   - Continues with real-time blocks

2. **Fast Sync** (For quick setup)
   ```yaml
   fast_sync: true
   listen_new_blocks: true
   start_height: 1000000  # Recent height
   ```
   - Downloads state snapshot at height
   - Skips historical block processing
   - Starts real-time indexing

3. **Real-time Only** (Ongoing monitoring)
   ```yaml
   parse_old_blocks: false
   listen_new_blocks: true
   start_height: 0  # Auto-uses latest DB height
   ```
   - Only indexes new blocks
   - Assumes historical data already indexed

4. **Catch-up Mode** (Fix gaps)
   ```yaml
   parse_old_blocks: true
   listen_new_blocks: false
   start_height: 1000
   ```
   - Only fills historical gaps
   - Doesn't monitor new blocks

### Performance Characteristics

**Throughput**:
- **Workers**: More workers = faster processing (diminishing returns after ~10)
- **Queue Size**: Larger buffer = fewer context switches
- **Average Block Time**: Shorter = more frequent polling, higher CPU
- **Database Connections**: Should match or exceed worker count

**Typical Performance**:
```
Workers: 5
Block Time: 2s
Processing Time: 0.5s/block (average)

Throughput: 
- Real-time: Can keep up with 2s blocks easily
- Historical: ~600 blocks/minute (5 workers × 0.5s = 2.5s per batch of 5)
- Catch-up time: 100,000 blocks ≈ 2.8 hours
```

### Genesis Processing

```
1. Read Genesis File
   ↓
2. Parse Genesis State
   ↓
3. Extract Module States:
   - Auth: Initial accounts
   - Bank: Initial balances
   - Staking: Initial validators
   - Gov: Initial params
   - etc.
   ↓
4. Store in Database
   ↓
5. Mark Genesis as Processed
```

### Periodic Operations

**Every 2 Minutes**:
- Update token prices (pricefeed)

**Every 10 Minutes**:
- Update total supply (bank)

**Every Hour**:
- Update community pool (distribution)
- Calculate voting power distribution (staking)

**Daily**:
- Update inflation metrics (mint)
- Calculate average delegation ratio (staking)
- Daily refetch operations

---

## Hasura Integration

### GraphQL API Layer

Hasura sits on top of the PostgreSQL database and automatically generates GraphQL APIs.

### Common Queries

**Get Recent Blocks**
```graphql
query {
  block(limit: 10, order_by: {height: desc}) {
    height
    hash
    num_txs
    timestamp
    proposer_address
  }
}
```

**Get Validator Info**
```graphql
query {
  validator {
    consensus_address
    validator_info {
      operator_address
      max_rate
    }
    validator_description {
      moniker
      identity
      website
    }
    validator_voting_power {
      voting_power
    }
  }
}
```

**Get Proposals**
```graphql
query {
  proposal(order_by: {id: desc}) {
    id
    title
    status
    submit_time
    voting_end_time
    proposal_tally_result {
      yes
      no
      abstain
      no_with_veto
    }
  }
}
```

### Hasura Actions

Custom business logic exposed via REST endpoints:

- Get account balance
- Get delegations and rewards
- Get unbonding delegations
- Get redelegations
- Get validator commission
- Get delegator withdraw address

---

## Monitoring & Observability

### Prometheus Metrics

Exposed on port 5000 (when enabled):

- `callisto_blocks_processed`: Total blocks processed
- `callisto_block_processing_time`: Block processing duration
- `callisto_transactions_processed`: Total transactions processed
- `callisto_errors_total`: Total errors encountered
- `callisto_current_height`: Current indexed height

### Logging

**Levels**: debug, info, warn, error

**Formats**: text (human-readable), json (structured)

**Key Log Events**:
- Block processing start/end
- Transaction parsing
- Module execution
- Database operations
- Error conditions

### Health Checks

- Database connection status
- Node RPC/gRPC connectivity
- Block sync status
- Latest indexed height vs chain height

---

## Development & Testing

### Testing

**Unit Tests**
```bash
make test-unit
```

Features:
- In-memory SQL database (ramsql)
- Mock RPC/gRPC clients
- Isolated module testing

**Test Coverage**
```bash
make test-cover
```

### Local Development

1. **Setup Local Database**
```bash
./setup-local-db.sh
```

2. **Run Migrations**
```bash
psql -U postgres -d bdjuno -f database/schema/schema.sql
```

3. **Start Callisto**
```bash
callisto start
```

### Docker Development

**Build Image**
```bash
docker build -t callisto .
```

**Run with Docker Compose**
```bash
docker-compose up
```

Services:
- Callisto parser
- PostgreSQL database
- Hasura GraphQL engine

---

## Performance Considerations

### Optimization Strategies

1. **Partitioning**
   - Transaction and message tables partitioned
   - Configurable partition size (default: 100,000 rows)
   - Reduces query time for recent data

2. **Batch Processing**
   - Configurable worker count
   - Parallel transaction processing
   - Batch inserts for messages

3. **Indexing**
   - Strategic indexes on foreign keys
   - GIN indexes for array searches
   - Height-based indexes for time queries

4. **Connection Pooling**
   - Configurable max open connections
   - Idle connection management
   - Connection reuse

5. **Caching**
   - In-memory caching of validator info
   - Recent block caching
   - Parameter caching

### Scalability

**Horizontal Scaling**:
- Run multiple Callisto instances for different height ranges
- Use partitioning for data distribution

**Vertical Scaling**:
- Increase worker count
- Allocate more database connections
- Increase memory for caching

---

## Deployment

### Production Deployment

1. **Prepare Configuration**
   - Set production RPC/gRPC endpoints
   - Configure database connection
   - Set appropriate worker count
   - Enable telemetry

2. **Database Setup**
   - Create PostgreSQL database
   - Run schema migrations
   - Set up backup strategy

3. **Deploy Callisto**
   - Build production binary or Docker image
   - Start with `callisto start`
   - Monitor logs and metrics

4. **Deploy Hasura**
   - Point Hasura to PostgreSQL
   - Configure metadata
   - Set up actions endpoints

5. **Monitoring**
   - Set up Prometheus scraping
   - Configure alerting
   - Set up log aggregation

### High Availability

- Run multiple Callisto instances with load balancing
- PostgreSQL replication for database HA
- Health check endpoints for load balancers

---

## Troubleshooting

### Common Issues

**1. Database Connection Errors**
- Check database URL in config
- Verify PostgreSQL is running
- Check firewall rules

**2. Block Sync Lagging**
- Increase worker count
- Check RPC endpoint performance
- Review database query performance

**3. Missing Data**
- Verify module is enabled in config
- Check for parsing errors in logs
- Verify RPC/gRPC endpoint compatibility

**4. High Memory Usage**
- Reduce worker count
- Decrease batch size
- Review module memory usage

### Debug Mode

Enable debug logging:
```yaml
logging:
  level: debug
  format: json
```

---

## Contributing

### Code Organization

```
callisto/
├── cmd/              # Command-line interface
├── database/         # Database layer and schema
├── modules/          # Blockchain modules
├── types/            # Type definitions
├── utils/            # Utility functions
└── tools/            # Development tools
```

### Adding a New Module

1. Create module directory under `modules/`
2. Implement module interfaces:
   - `Module`: Basic module interface
   - `GenesisModule`: Genesis processing
   - `BlockModule`: Per-block processing
   - `MessageModule`: Message handling
   - `PeriodicOperationsModule`: Scheduled tasks
3. Add database schema in `database/schema/`
4. Register module in `modules/registrar.go`
5. Add tests

---

## Resources

### Documentation
- [Juno Documentation](https://docs.juno.zone)
- [Big Dipper Docs](https://docs.bigdipper.live)
- [Cosmos SDK](https://docs.cosmos.network)

### Repository
- **GitHub**: [forbole/callisto](https://github.com/forbole/callisto)
- **License**: See LICENSE file

---

## Glossary

- **Callisto**: The blockchain indexer (formerly BDJuno)
- **Juno**: The underlying indexing framework
- **Big Dipper**: The blockchain explorer UI
- **Hasura**: GraphQL engine for data access
- **Cosmos SDK**: Blockchain framework
- **CometBFT**: Consensus engine (formerly Tendermint)
- **Module**: Independent functional unit handling specific blockchain features
- **Partition**: Database table splitting for performance
- **Pre-commit**: Validator vote on a block

---

## Summary

Callisto is a comprehensive blockchain indexing solution that:
- Parses Cosmos SDK blockchain data in real-time
- Stores structured data in PostgreSQL
- Supports 18+ blockchain modules
- Provides GraphQL APIs via Hasura
- Offers monitoring and observability
- Scales horizontally and vertically
- Integrates seamlessly with Big Dipper

It's production-ready, highly configurable, and designed for reliability and performance in blockchain data indexing.
