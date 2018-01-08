package iprs_resolver

import (
	"bytes"
	"context"
	"fmt"
	"time"

	rsp "github.com/dirkmc/go-iprs/path"
	nspb "github.com/ipfs/go-ipfs/namesys/pb"
	routing "gx/ipfs/QmPCGUjMRuBcPybZFpjhzpifwPP9wPRoiy5geTQKU4vqWA/go-libp2p-routing"
	u "gx/ipfs/QmPsAfmDBnZN3kZGSuNwvCNDZiHneERSKmRcFyG3UkvcT3/go-ipfs-util"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	ci "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
)

const DefaultIpnsCacheTTL = time.Minute

type IpnsResolver struct {
	vstore      routing.ValueStore
	cache       *ResolverCache
}

// cachesize is the limit of the number of entries in the lru cache. Setting it
// to '0' will disable caching.
func NewIpnsResolver(vs routing.ValueStore, cachesize int, ttlp *time.Duration) *IpnsResolver {
	ttl := DefaultIpnsCacheTTL
	tp := &ttl
	if ttlp != nil {
		tp = &ttl
	}

	rs := IpnsResolver{vstore: vs}
	rs.cache = NewResolverCache(&rs, cachesize, tp)
	return &rs
}

func (r *IpnsResolver) Resolve(ctx context.Context, iprsKey rsp.IprsPath) (string, error) {
	log.Debugf("IPNS Resolve %s", iprsKey)

	// Use the routing system to get the entry
	val, err := r.cache.GetValue(ctx, iprsKey.String())
	if err != nil {
		log.Warningf("IpnsResolver get failed for %s", iprsKey)
		return "", err
	}

	return string(val), nil
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
	err = r.checkValue(val)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to parse IPNS record target [%s] at %s: %s", entry.GetValue(), iprsKey, err)
	}

	return val, eol, nil
}

func (r *IpnsResolver) checkValue([]byte) error {
	// TODO:
	// Value must be
	// - /ipfs/<hash>
	// - /ipns/<hash>
	// - /ipns/<domain>
	return nil
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
