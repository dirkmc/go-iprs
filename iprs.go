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
	path "github.com/ipfs/go-ipfs/path"
	logging "github.com/ipfs/go-log"
	routing "gx/ipfs/QmPR2JzfKd9poHx9XBhzoFeBBC31ZM3W5iUPKJZWyaoZZm/go-libp2p-routing"
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

func NewNameSystem(r routing.ValueStore, ds ds.Datastore, cachesize int) NameSystem {
	factory := rec.NewRecordFactory(r)
	seqm := psh.NewSeqManager(ds, r)
	return &mpns{
		resolvers: map[string]rsv.Lookup{
			"dns":      rsv.NewDNSResolver(),
			"proquint": new(rsv.ProquintResolver),
			"dht":      rsv.NewDHTResolver(r, factory, cachesize),
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
	err := ns.publishers["/iprs/"].Publish(ctx, iprsKey, record)
	if err != nil {
		return err
	}
	// TODO: What is the idea of this call? Don't get what it's doing
	//ns.addToDHTCache(name, value, time.Now().Add(DefaultRecordTTL))
	return nil
}

/*
func (ns *mpns) PublishWithEOL(ctx context.Context, name ci.PrivKey, value path.Path, eol time.Time) error {
	err := ns.publishers["/iprs/"].PublishWithEOL(ctx, name, value, eol)
	if err != nil {
		return err
	}
	ns.addToDHTCache(name, value, eol)
	return nil
}

func (ns *mpns) addToDHTCache(key ci.PrivKey, value path.Path, eol time.Time) {
	rr, ok := ns.resolvers["dht"].(*routingResolver)
	if !ok {
		// should never happen, purely for sanity
		log.Panicf("unexpected type %T as DHT resolver.", ns.resolvers["dht"])
	}
	if rr.cache == nil {
		// resolver has no caching
		return
	}

	var err error
	value, err = path.ParsePath(value.String())
	if err != nil {
		log.Error("could not parse path")
		return
	}

	name, err := peer.IDFromPrivateKey(key)
	if err != nil {
		log.Error("while adding to cache, could not get peerid from private key")
		return
	}

	if time.Now().Add(DefaultResolverCacheTTL).Before(eol) {
		eol = time.Now().Add(DefaultResolverCacheTTL)
	}
	rr.cache.Add(name.Pretty(), cacheEntry{
		val: value,
		eol: eol,
	})
}
*/
