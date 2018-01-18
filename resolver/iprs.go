package iprs_resolver

import (
	"context"
	"fmt"
	"strings"
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
	parent      *Resolver
	vstore      routing.ValueStore
	dag         node.NodeGetter
	cache       *ResolverCache
	verifier    *rec.MasterRecordVerifier
}

func NewIprsResolver(parent *Resolver, vs routing.ValueStore, dag node.NodeGetter, opts *CacheOpts) *IprsResolver {
	if opts == nil {
		ttl := DefaultIprsCacheTTL
		opts = &CacheOpts{10, &ttl}
	}
	v := rec.NewMasterRecordVerifier(dag)
	rs := IprsResolver{parent: parent, vstore: vs, dag: dag, verifier: v}
	rs.cache = NewResolverCache(&rs, opts)
	return &rs
}

func (r *IprsResolver) Accept(p string) bool {
	parts := strings.Split(p, "/")
	if len(parts) < 4 {
		return false
	}
	if parts[1] != "iprs" {
		return false
	}
	_, err := cid.Decode(parts[2])
	return err == nil
}

func (r *IprsResolver) Resolve(ctx context.Context, p string) (string, []string, error) {
	log.Debugf("IPRS Resolve %s", p)

	if !r.Accept(p) {
		return "", nil, fmt.Errorf("IPRS resolver cannot resolve %s", p)
	}
	parts := strings.Split(p, "/")

	// Use the routing system to get the entry
	k := "/iprs/" + parts[2] + "/" + parts[3]
	val, err := r.cache.GetValue(ctx, k)
	if err != nil {
		log.Warningf("IprsResolver get failed for %s", k)
		return "", nil, err
	}

	log.Debugf("IPRS Resolve %s successful", k)
	return string(val), parts[4:], nil
}

func (r *IprsResolver) GetValue(ctx context.Context, k string) ([]byte, *time.Time, error) {
	iprsKey, err := rsp.FromString(k)
	if err != nil {
		log.Warningf("Failed to parse IPRS record path %s", k)
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
	val := record.Value
	if !r.parent.IsResolvable(string(val)) {
		return nil, nil, fmt.Errorf("Failed to parse IPRS record target [%s] at %s", val, iprsKey)
	}

	return val, eol, nil
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
