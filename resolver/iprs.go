package iprs_resolver

import (
	"context"
	"fmt"
	"time"

	ld "github.com/dirkmc/go-iprs/ipld"
	rsp "github.com/dirkmc/go-iprs/path"
	rec "github.com/dirkmc/go-iprs/record"
	node "gx/ipfs/QmNwUEK7QbwSqyKBu3mMtToo8SUc6wQJ7gdZq4gGGJqfnf/go-ipld-format"
	routing "gx/ipfs/QmPCGUjMRuBcPybZFpjhzpifwPP9wPRoiy5geTQKU4vqWA/go-libp2p-routing"
	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
)

const DefaultIprsCacheTTL = time.Minute

type IprsResolver struct {
	vstore      routing.ValueStore
	dag         node.NodeGetter
	cache       *ResolverCache
	verifier    *rec.MasterRecordVerifier
}

// cachesize is the limit of the number of entries in the lru cache. Setting it
// to '0' will disable caching.
func NewIprsResolver(vs routing.ValueStore, dag node.NodeGetter, cachesize int, ttlp *time.Duration) *IprsResolver {
	ttl := DefaultIprsCacheTTL
	tp := &ttl
	if ttlp != nil {
		tp = &ttl
	}
	v := rec.NewMasterRecordVerifier(dag)

	rs := IprsResolver{vstore: vs, dag: dag, verifier: v}
	rs.cache = NewResolverCache(&rs, cachesize, tp)
	return &rs
}

func (r *IprsResolver) Resolve(ctx context.Context, iprsKey rsp.IprsPath) (*cid.Cid, error) {
	log.Debugf("IPRS Resolve %s", iprsKey)

	// Use the routing system to get the entry
	val, err := r.cache.GetValue(ctx, iprsKey.String())
	if err != nil {
		log.Warningf("IprsResolver get failed for %s", iprsKey)
		return nil, err
	}

	return cid.Parse(val)
}

func (r *IprsResolver) GetValue(ctx context.Context, k string) ([]byte, *time.Time, error) {
	iprsKey, err := rsp.FromString(k)
	if err != nil {
		return nil, nil, err
	}

	// Retrieve record from the value store
	b, err := r.vstore.GetValue(ctx, k)
	if err != nil {
		log.Warningf("Failed to retrieve IPRS record %s from value store", iprsKey)
		return nil, nil, err
	}

	// Unmarshall into an IPRS record CID
	iprsCid, err := cid.Cast(b)
	if err != nil {
		log.Warningf("Failed to unmarshal IPRS record at %s", iprsKey)
		return nil, nil, err
	}

	// Retrieve node from the block store
	n, err := r.dag.Get(ctx, iprsCid)
	if err != nil {
		log.Warningf("Failed to retrieve IPRS record %s with CID %s from block store", iprsKey, iprsCid)
		return nil, nil, err
	}
	iprsNode, err := ld.DecodeIprsBlock(n)
	if err != nil {
		log.Warningf("Failed to decode IPRS record %s with CID %s from block format", iprsKey, iprsCid)
		return nil, nil, err
	}
	record := rec.NewRecordFromNode(iprsNode)

	// Verify record signatures etc are correct
	log.Debugf("Verifying IPRS record %s with CID %s", iprsKey, iprsCid)
	err = r.verifier.Verify(ctx, iprsKey, record)
	if err != nil {
		log.Warningf("Failed to verify IPRS record at %s", iprsKey)
		return nil, nil, err
	}

	eol := r.getEol(record)
	val := record.Value.Bytes()
	err = r.checkValue(val)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to parse IPRS record target [%s] at %s: %s", val, iprsKey, err)
	}

	return val, eol, nil
}

func (r *IprsResolver) checkValue([]byte) error {
	// TODO:
	// Value must be
	// - /iprs/<cid>
	return nil
}

func (r *IprsResolver) getEol(record *rec.Record) *time.Time {
	// If it's an EOL record, it's just the EOL
	if record.Validity.ValidationType == ld.ValidationType_EOL {
		eol, err := rec.EolParseValidation(record)
		if err != nil {
			return nil
		}
		return &eol
	}
	// If it's a TimeRange record, it's the end time
	// (note that a nil end time means infinity)
	if record.Validity.ValidationType == ld.ValidationType_TimeRange {
		rng, err := rec.RangeParseValidation(record)
		if err != nil || rng[1] == nil {
			return nil
		}
		return rng[1]
	}
	return nil
}
