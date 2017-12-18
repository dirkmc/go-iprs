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
	mh "gx/ipfs/QmYeKnKpubCMRiq3PGZcTREErthbb5Q9cXsCoSkD9bjEBd/go-multihash"
	isd "gx/ipfs/QmZmmuAXgX73UQmX1jRKjTGmjzq24Jinqkq8vzkBtno4uX/go-is-domain"
)

var log = logging.Logger("iprs")

const DefaultRecordTTL = 24 * time.Hour
const DefaultResolverCacheTTL = time.Minute

// mprs (a multi-protocol NameSystem) implements generic IPFS naming.
//
// Uses several Resolvers:
// (a) IPFS routing naming: SFS-like PKI names.
// (b) dns domains: resolves using links in DNS TXT records
// (c) proquints: interprets string as the raw byte data.
//
// It can only publish to: (a) IPFS routing naming.
//

type mprs struct {
	resolvers  map[string]rsv.Lookup
	publishers map[string]Publisher
}

func NewRecordSystem(vstore vs.ValueStore, cachesize int) RecordSystem {
	factory := rec.NewRecordFactory(vstore)
	seqm := psh.NewSeqManager(vstore)
	cachedvs := vs.NewCachedValueStore(vstore, cachesize, nil)
	return &mprs{
		resolvers: map[string]rsv.Lookup{
			"dns":      rsv.NewDNSResolver(),
			"proquint": new(rsv.ProquintResolver),
			"dht":      rsv.NewDHTResolver(cachedvs, factory),
		},
		publishers: map[string]Publisher{
			"/iprs/": psh.NewDHTPublisher(seqm),
		},
	}
}

// Resolve implements Resolver.
func (ns *mprs) Resolve(ctx context.Context, name string) (path.Path, error) {
	return ns.ResolveN(ctx, name, rsv.DefaultDepthLimit)
}

// ResolveN implements Resolver.
func (ns *mprs) ResolveN(ctx context.Context, name string, depth int) (path.Path, error) {
	if strings.HasPrefix(name, "/ipfs/") {
		return path.ParsePath(name)
	}

	if !strings.HasPrefix(name, "/") {
		return path.ParsePath("/ipfs/" + name)
	}

	return rsv.Resolve(ctx, ns, name, depth, "/iprs/", "/ipns/")
}

// ResolveOnce implements Lookup.
func (ns *mprs) ResolveOnce(ctx context.Context, name string) (string, error) {
	log.Debugf("RecordSystem ResolveOnce %s", name)

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
		log.Debugf("RecordSystem.ResolveOnce DHT resolve %s", key)
		return resolveOnce("dht", key)
	}

	if isd.IsDomain(key) {
		log.Debugf("RecordSystem.ResolveOnce DNS resolve %s", key)
		return resolveOnce("dns", key)
	}

	log.Debugf("RecordSystem.ResolveOnce proquint resolve %s", key)
	return resolveOnce("proquint", key)
}

// Publish implements Publisher
func (ns *mprs) Publish(ctx context.Context, iprsKey rsp.IprsPath, record *r.Record) error {
	return ns.publishers["/iprs/"].Publish(ctx, iprsKey, record)
}
