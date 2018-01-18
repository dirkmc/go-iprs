package iprs_resolver

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
	rsp "github.com/dirkmc/go-iprs/path"
	nspb "github.com/ipfs/go-ipfs/namesys/pb"
	routing "gx/ipfs/QmPCGUjMRuBcPybZFpjhzpifwPP9wPRoiy5geTQKU4vqWA/go-libp2p-routing"
	u "gx/ipfs/QmPsAfmDBnZN3kZGSuNwvCNDZiHneERSKmRcFyG3UkvcT3/go-ipfs-util"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	ci "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
)

const DefaultIpnsCacheTTL = time.Minute

type IpnsResolver struct {
	parent      *Resolver
	vstore      routing.ValueStore
	cache       *ResolverCache
}

func NewIpnsResolver(parent *Resolver, vs routing.ValueStore, opts *CacheOpts) *IpnsResolver {
	if opts == nil {
		ttl := DefaultIpnsCacheTTL
		opts = &CacheOpts{10, &ttl}
	}
	rs := IpnsResolver{parent: parent, vstore: vs}
	rs.cache = NewResolverCache(&rs, opts)
	return &rs
}

func (r *IpnsResolver) Accept(p string) bool {
	parts := strings.Split(p, "/")
	if len(parts) < 3 {
		return false
	}
	if parts[1] != "ipns" {
		return false
	}
	_, err := cid.Decode(parts[2])
	return err == nil
}

func (r *IpnsResolver) Resolve(ctx context.Context, p string) (string, []string, error) {
	log.Debugf("IPNS Resolve %s", p)

	if !r.Accept(p) {
		return "", nil, fmt.Errorf("IPNS resolver cannot resolve %s", p)
	}
	parts := strings.Split(p, "/")

	// Use the routing system to get the entry
	k := "/ipns/" + parts[2]
	val, err := r.cache.GetValue(ctx, k)
	if err != nil {
		log.Warningf("IpnsResolver get failed for %s", k)
		return "", nil, err
	}

	log.Debugf("IPNS Resolve %s => %s", k, val)
	return string(val), parts[3:], nil
}

func (r *IpnsResolver) GetValue(ctx context.Context, k string) ([]byte, *time.Time, error) {
	iprsKey, err := rsp.FromString(k)
	if err != nil {
		return nil, nil, err
	}

	var entry *nspb.IpnsEntry
	var pubkey ci.PubKey

	// Get the IPNS record and the public key in parallel
	resp := make(chan error, 2)
	go func() {
		// IPNS records are stored in the DHT at /ipns/string(<hash>)
		// ie the hash is not B58 encoded
		name := "/ipns/" + string(iprsKey.Cid().Hash())
		val, err := r.vstore.GetValue(ctx, name)
		if err != nil {
			log.Debugf("RoutingResolver: dht get %s failed: %s", name, err)
			resp <- err
			return
		}

		entry = new(nspb.IpnsEntry)
		err = proto.Unmarshal(val, entry)
		if err != nil {
			resp <- err
			return
		}

		resp <- nil
	}()

	go func() {
		pubk, err := routing.GetPublicKey(r.vstore, ctx, iprsKey.Cid().Bytes())
		if err != nil {
			resp <- err
			return
		}

		pubkey = pubk
		resp <- nil
	}()

	for i := 0; i < 2; i++ {
		err = <-resp
		if err != nil {
			return nil, nil, err
		}
	}

	// Check signature with public key
	if ok, err := pubkey.Verify(r.entryDataForSig(entry), entry.GetSignature()); err != nil || !ok {
		return nil, nil, fmt.Errorf("Failed to verify IPNS record at %s: invalid signature", iprsKey)
	}

	eol := r.getEol(entry)
	val := entry.GetValue()
	if !r.parent.IsResolvable(string(val)) {
		return nil, nil, fmt.Errorf("Failed to parse IPNS record target [%s] at %s", val, iprsKey)
	}

	return val, eol, nil
}

func (r *IpnsResolver) entryDataForSig(e *nspb.IpnsEntry) []byte {
	return bytes.Join([][]byte{
		e.Value,
		e.Validity,
		[]byte(fmt.Sprint(e.GetValidityType())),
	},
		[]byte{})
}

func (r *IpnsResolver) getEol(e *nspb.IpnsEntry) *time.Time {
	if e.GetValidityType() == nspb.IpnsEntry_EOL {
		eol, err := u.ParseRFC3339(string(e.GetValidity()))
		if err != nil {
			return nil
		}
		return &eol
	}
	return nil
}
