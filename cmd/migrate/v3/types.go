package v3

import (
	v3 "github.com/1119-Labs/callisto/v4/lib/cmd/migrate/v3"

	"github.com/1119-Labs/callisto/v4/modules/actions"
)

type Config struct {
	v3.Config `yaml:"-,inline"`

	// The following are there to support modules which config are present if they are enabled

	Actions *actions.Config `yaml:"actions"`
}
