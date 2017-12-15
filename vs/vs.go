package iprs_vs

import (
	"context"

	routing "gx/ipfs/QmPR2JzfKd9poHx9XBhzoFeBBC31ZM3W5iUPKJZWyaoZZm/go-libp2p-routing"
)

type ValueStore interface {
	routing.ValueStore

	// GetLocalValue only checks the local data store for value corresponding to given key
	// (it doesn't got out to the network)
	GetLocalValue(context.Context, string) ([]byte, error)
}
