package bank

import (
	parsecmdtypes "github.com/1119-Labs/callisto/v4/lib/cmd/parse/types"
	"github.com/spf13/cobra"
)

// NewBankCmd returns the Cobra command allowing to fix various things related to the x/bank module
func NewBankCmd(parseConfig *parsecmdtypes.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bank",
		Short: "Fix things related to the x/bank module",
	}

	cmd.AddCommand(
		supplyCmd(parseConfig),
	)

	return cmd
}
