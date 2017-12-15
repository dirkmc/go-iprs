package iprs

import (
	"context"
	"strings"
	"time"

	rsp "github.com/dirkmc/go-iprs/path"
	psh "github.com/dirkmc/go-iprs/publisher"
	r "github.com/dirkmc/go-iprs/record"
	rec "github.com/dirkmc/go-iprs/record"
	rsv "github.com/dirkmc/go-iprs/resolver"
	vs "github.com/dirkmc/go-iprs/vs"
	path "github.com/ipfs/go-ipfs/path"
	logging "github.com/ipfs/go-log"
	mh "gx/ipfs/QmU9a9NV9RdPNwZQDYd5uKsm6N6LJLSvLbywDDYFbaaC6P/go-multihash"
	isd "gx/ipfs/QmZmmuAXgX73UQmX1jRKjTGmjzq24Jinqkq8vzkBtno4uX/go-is-domain"
	ds "gx/ipfs/QmdHG8MAuARdGHxx4rPQASLcvhz24fzjSQq7AJRAQEorq5/go-datastore"
)

var log = logging.Logger("iprs")

const DefaultRecordTTL = 24 * time.Hour

// mpns (a multi-protocol NameSystem) implements generic IPFS naming.
//
// Uses several Resolvers:
// (a) IPFS routing naming: SFS-like PKI names.
// (b) dns domains: resolves using links in DNS TXT records
// (c) proquints: interprets string as the raw byte data.
//
// It can only publish to: (a) IPFS routing naming.
//

type mpns struct {
	resolvers  map[string]rsv.Lookup
	publishers map[string]Publisher
}

func NewNameSystem(vstore vs.ValueStore, ds ds.Datastore, cachesize int) NameSystem {
	factory := rec.NewRecordFactory(vstore)
	seqm := psh.NewSeqManager(ds, vstore)
	return &mpns{
		resolvers: map[string]rsv.Lookup{
			"dns":      rsv.NewDNSResolver(),
			"proquint": new(rsv.ProquintResolver),
			"dht":      rsv.NewDHTResolver(vs.NewCachedValueStore(vstore, cachesize, nil), factory),
		},
		publishers: map[string]Publisher{
			"/iprs/": psh.NewDHTPublisher(seqm),
		},
	}
}

const DefaultResolverCacheTTL = time.Minute

// Resolve implements Resolver.
func (ns *mpns) Resolve(ctx context.Context, name string) (path.Path, error) {
	return ns.ResolveN(ctx, name, rsv.DefaultDepthLimit)
}

// ResolveN implements Resolver.
func (ns *mpns) ResolveN(ctx context.Context, name string, depth int) (path.Path, error) {
	if strings.HasPrefix(name, "/ipfs/") {
		return path.ParsePath(name)
	}

	if !strings.HasPrefix(name, "/") {
		return path.ParsePath("/ipfs/" + name)
	}

	return rsv.Resolve(ctx, ns, name, depth, "/iprs/", "/ipns/")
}

// ResolveOnce implements Lookup.
func (ns *mpns) ResolveOnce(ctx context.Context, name string) (string, error) {
	if !strings.HasPrefix(name, "/iprs/") && !strings.HasPrefix(name, "/ipns/") {
		name = "/iprs/" + name
	}

	segments := strings.SplitN(name, "/", 4)
	if len(segments) < 3 || segments[0] != "" {
		log.Warningf("Invalid name syntax for %s", name)
		return "", rsv.ErrResolveFailed
	}

	resolveOnce := func(rname string, key string) (string, error) {
		res, ok := ns.resolvers[rname]
		if !ok {
			log.Warningf("Could not find resolver with name %s", rname)
			return "", rsv.ErrResolveFailed
		}
		p, err := res.ResolveOnce(ctx, key)
		if err != nil {
			log.Warningf("Could not resolve with %s resolver: %s", rname, err)
			return "", rsv.ErrResolveFailed
		}

		if len(segments) > 3 {
			return strings.TrimRight(p, "/") + "/" + segments[3], nil
		}
		return p, nil
	}

	// Resolver selection:
	// 1. if it is a multihash resolve through "dht"
	// 2. if it is a domain name, resolve through "dns"
	// 3. otherwise resolve through the "proquint" resolver
	key := segments[2]

	_, err := mh.FromB58String(key)
	if err == nil {
		return resolveOnce("dht", key)
	}

	if isd.IsDomain(key) {
		return resolveOnce("dns", key)
	}

	return resolveOnce("proquint", key)
}

// Publish implements Publisher
func (ns *mpns) Publish(ctx context.Context, iprsKey rsp.IprsPath, record *r.Record) error {
	return ns.publishers["/iprs/"].Publish(ctx, iprsKey, record)
}
