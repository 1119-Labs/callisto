package remote

import (
	"context"

	"google.golang.org/grpc"

	"github.com/1119-Labs/callisto/v4/lib/node"
)

var (
	_ node.Source = &Source{}
)

// Source implements the keeper.Source interface relying on a GRPC connection
type Source struct {
	Ctx      context.Context
	GrpcConn *grpc.ClientConn
}

// NewSource returns a new Source instance
func NewSource(config *GRPCConfig) (*Source, error) {
	return &Source{
		Ctx:      context.Background(),
		GrpcConn: MustCreateGrpcConnection(config),
	}, nil
}

// Type implements keeper.Type
func (k Source) Type() string {
	return node.RemoteKeeper
}
