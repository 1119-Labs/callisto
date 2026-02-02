package mint

import (
	"fmt"

	parsecmdtypes "github.com/1119-Labs/callisto/v4/lib/cmd/parse/types"
	"github.com/1119-Labs/callisto/v4/lib/types/config"
	"github.com/spf13/cobra"

	"github.com/1119-Labs/callisto/v4/database"
	"github.com/1119-Labs/callisto/v4/modules/mint"
	modulestypes "github.com/1119-Labs/callisto/v4/modules/types"
	"github.com/1119-Labs/callisto/v4/utils"
)

// inflationCmd returns the Cobra command allowing to refresh x/mint inflation
func inflationCmd(parseConfig *parsecmdtypes.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "inflation",
		Short: "Refresh inflation",
		RunE: func(cmd *cobra.Command, args []string) error {
			parseCtx, err := parsecmdtypes.GetParserContext(config.Cfg, parseConfig)
			if err != nil {
				return err
			}

			cdc := utils.GetCodec()
			sources, err := modulestypes.BuildSources(config.Cfg.Node, cdc)
			if err != nil {
				return err
			}

			// Get the database
			db := database.Cast(parseCtx.Database)

			// Build mint module
			mintModule := mint.NewModule(sources.MintSource, cdc, db)

			err = mintModule.UpdateInflation()
			if err != nil {
				return fmt.Errorf("error while updating inflation: %s", err)
			}

			return nil
		},
	}
}
