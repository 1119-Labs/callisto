package types

// HeightQueue represents a queue for block heights.
// Implementations can provide in-memory or external queue backends.
type HeightQueue interface {
	// Publish enqueues a block height.
	Publish(height int64) error
	// Consume starts consuming block heights and invokes the handler.
	// If the handler returns an error, the message should be re-queued.
	Consume(handler func(height int64) error) error
	// Close closes any underlying resources.
	Close() error
}

// TxQueue represents a queue for transaction hashes.
// Used to decouple block processing from transaction processing.
type TxQueue interface {
	// Publish enqueues a transaction hash along with its block height.
	Publish(txHash string, height int64) error
	// Consume starts consuming transaction hashes and invokes the handler.
	// If the handler returns an error, the message should be re-queued.
	Consume(handler func(txHash string, height int64) error) error
	// Close closes any underlying resources.
	Close() error
}
