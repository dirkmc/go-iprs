package iprs

import (
	"context"
	"time"

	rsp "github.com/dirkmc/go-iprs/path"
	psh "github.com/dirkmc/go-iprs/publisher"
	r "github.com/dirkmc/go-iprs/record"
	routing "gx/ipfs/QmPCGUjMRuBcPybZFpjhzpifwPP9wPRoiy5geTQKU4vqWA/go-libp2p-routing"
	rsv "github.com/dirkmc/go-iprs/resolver"
	mdag "github.com/ipfs/go-ipfs/merkledag"
	node "gx/ipfs/QmNwUEK7QbwSqyKBu3mMtToo8SUc6wQJ7gdZq4gGGJqfnf/go-ipld-format"
	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
)

var log = logging.Logger("iprs")

const DefaultRecordTTL = 24 * time.Hour
const DefaultResolverCacheTTL = time.Minute

// mprs (a multi-protocol RecordSystem) implements generic IPFS naming.
//
// Uses several Resolvers:
// (a) IPRS: IPFS record system.
// (b) IPNS: IPFS routing naming - SFS-like PKI names.
// (c) dns domains: resolves using links in DNS TXT records
//
// It can only publish to: (a) IPFS record system.
//

type mprs struct {
	resolver  *rsv.Resolver
	publisher Publisher
}

func NewRecordSystem(vstore routing.ValueStore, dag mdag.DAGService, cachesize int) RecordSystem {
	resolver := rsv.NewResolver(vstore, dag, cachesize, nil)
	publisher := psh.NewDHTPublisher(vstore, dag)
	return &mprs{resolver, publisher}
}

// Resolve implements Resolver.
func (rs *mprs) Resolve(ctx context.Context, name string) (*node.Link, []string, error) {
	return rs.ResolveN(ctx, name, rsv.DefaultDepthLimit)
}

// ResolveN implements Resolver.
func (rs *mprs) ResolveN(ctx context.Context, name string, depth int) (*node.Link, []string, error) {
	return rs.resolver.Resolve(ctx, name, depth)
}

// Publish implements Publisher
func (rs *mprs) Publish(ctx context.Context, iprsKey rsp.IprsPath, record *r.Record) error {
	return rs.publisher.Publish(ctx, iprsKey, record)
}
