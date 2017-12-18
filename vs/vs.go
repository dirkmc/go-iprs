package iprs_vs

import (
	"context"

	routing "gx/ipfs/QmPCGUjMRuBcPybZFpjhzpifwPP9wPRoiy5geTQKU4vqWA/go-libp2p-routing"
)

type ValueStore interface {
	routing.ValueStore

	// GetLocalValue only checks the local data store for value corresponding to given key
	// (it doesn't got out to the network)
	GetLocalValue(context.Context, string) ([]byte, error)
}
