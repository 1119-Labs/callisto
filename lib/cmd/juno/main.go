package main

import (
	"os"

	"github.com/1119-Labs/callisto/v4/lib/cmd/parse/types"

	"github.com/1119-Labs/callisto/v4/lib/modules/messages"
	"github.com/1119-Labs/callisto/v4/lib/modules/registrar"

	"github.com/1119-Labs/callisto/v4/lib/cmd"
)

func main() {
	// JunoConfig the runner
	config := cmd.NewConfig("juno").
		WithParseConfig(types.NewConfig().
			WithRegistrar(registrar.NewDefaultRegistrar(
				messages.CosmosMessageAddressesParser,
			)),
		)

	// Run the commands and panic on any error
	exec := cmd.BuildDefaultExecutor(config)
	err := exec.Execute()
	if err != nil {
		os.Exit(1)
	}
}
