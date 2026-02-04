package messages

import (
	"github.com/1119-Labs/callisto/v4/lib/database"
	"github.com/1119-Labs/callisto/v4/lib/types"
)

// HandleMsg represents a message handler that stores the given message inside the proper database table
func HandleMsg(
	index int, msg types.Message, tx *types.Transaction,
	parseAddresses MessageAddressesParser, db database.Database,
) error {

	// Get the involved addresses
	addresses, err := parseAddresses(tx)
	if err != nil {
		return err
	}

	return db.SaveMessage(int64(tx.Height), tx.TxHash, msg, addresses)
}
