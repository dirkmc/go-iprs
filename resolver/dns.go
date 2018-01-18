package iprs_resolver

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	isd "gx/ipfs/QmZmmuAXgX73UQmX1jRKjTGmjzq24Jinqkq8vzkBtno4uX/go-is-domain"
	path "github.com/ipfs/go-ipfs/path"
)

const DefaultDnsCacheTTL = time.Minute

type LookupTXTFunc func(name string) (txt []string, err error)

// DNSResolver implements a Resolver on DNS domains
type DNSResolver struct {
	parent      *Resolver
	cache       *ResolverCache
	lookupTXT LookupTXTFunc
}

// NewDNSResolver constructs a name resolver using DNS TXT records.
func NewDNSResolver(parent *Resolver, opts *CacheOpts) *DNSResolver {
	if opts == nil {
		ttl := DefaultDnsCacheTTL
		opts = &CacheOpts{10, &ttl}
	}
	rs := DNSResolver{parent: parent, lookupTXT: net.LookupTXT}
	rs.cache = NewResolverCache(&rs, opts)
	return &rs
}

type lookupRes struct {
	path  string
	error error
}

func (r *DNSResolver) Accept(p string) bool {
	parts := strings.Split(p, "/")
	return len(parts) > 2 && isd.IsDomain(parts[2])
}

func (r *DNSResolver) Resolve(ctx context.Context, p string) (string, []string, error) {
	log.Debugf("DNS Resolve %s", p)

	if !r.Accept(p) {
		return "", nil, fmt.Errorf("DNS resolver cannot resolve %s", p)
	}
	parts := strings.Split(p, "/")
	domain := parts[2]

	val, err := r.cache.GetValue(ctx, domain)
	if err != nil {
		log.Warningf("DnsResolver get failed for %s", domain)
		return "", nil, err
	}

	log.Debugf("DNS Resolve %s => %s", domain, val)
	return string(val), parts[3:], nil
}

func (r *DNSResolver) GetValue(ctx context.Context, domain string) ([]byte, *time.Time, error) {
	log.Debugf("DNSResolver resolving %s", domain)

	rootChan := make(chan lookupRes, 1)
	go workDomain(r, domain, rootChan)

	subChan := make(chan lookupRes, 1)
	go workDomain(r, "_dnslink."+domain, subChan)

	var subRes lookupRes
	select {
	case subRes = <-subChan:
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}

	if subRes.error == nil {
		return []byte(subRes.path), nil, nil
	}

	var rootRes lookupRes
	select {
	case rootRes = <-rootChan:
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}
	if rootRes.error == nil {
		return []byte(rootRes.path), nil, nil
	}

	return nil, nil, ErrResolveFailed
}

func workDomain(r *DNSResolver, name string, res chan lookupRes) {
	txt, err := r.lookupTXT(name)
	if err != nil {
		// Error is != nil
		res <- lookupRes{"", err}
		return
	}

	log.Debugf("DNSResolver lookupTXT(%s) => %s", name, txt)
	for _, t := range txt {
		p, err := r.parseEntry(t)
		if err == nil {
			res <- lookupRes{p, nil}
			return
		}
		log.Debugf("Could not parse entry %s", t)
	}
	res <- lookupRes{"", ErrResolveFailed}
}

func (r *DNSResolver) parseEntry(txt string) (string, error) {
	p, err := path.ParseCidToPath(txt) // bare IPFS multihashes
	if err == nil {
		return p.String(), nil
	}

	if r.parent.IsResolvable(txt) {
		return txt, nil
	}

	return r.tryParseDnsLink(txt)
}

// Parse links of the form
// dnslink=/ipfs/somepath
func (r *DNSResolver) tryParseDnsLink(txt string) (string, error) {
	parts := strings.SplitN(txt, "=", 2)
	if len(parts) == 2 && parts[0] == "dnslink" && r.parent.IsResolvable(parts[1]) {
		return parts[1], nil
	}

	return "", fmt.Errorf("Not a valid dnslink entry: %s", txt)
}
/*
// Must be of the form
// /iprs/<cid>
// /iprs/www.example.com
func isIprsPath(txt string) bool {
	parts := strings.Split(txt, "/")

	if len(parts) < 3 {
		return false
	}
	if parts[0] != "" {
		return false
	}
	if parts[1] != "iprs" {
		return false
	}
	if isd.IsDomain(parts[2]) {
		return true
	}

	_, err := cid.Parse(parts[2])
	return err == nil
}
*/