package iprs_resolver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	path "github.com/ipfs/go-ipfs/path"
	isd "gx/ipfs/QmZmmuAXgX73UQmX1jRKjTGmjzq24Jinqkq8vzkBtno4uX/go-is-domain"
	u "github.com/ipfs/go-ipfs-util"
)

type LookupTXTFunc func(name string) (txt []string, err error)

// DNSResolver implements a Resolver on DNS domains
type DNSResolver struct {
	lookupTXT LookupTXTFunc
	// TODO: maybe some sort of caching?
	// cache would need a timeout
}

// NewDNSResolver constructs a name resolver using DNS TXT records.
func NewDNSResolver() *DNSResolver {
	return &DNSResolver{lookupTXT: net.LookupTXT}
}

// newDNSResolver constructs a name resolver using DNS TXT records,
// returning a Lookup instead of NewDNSResolver's Resolver.
func newDNSResolver() Lookup {
	return &DNSResolver{lookupTXT: net.LookupTXT}
}

// Resolve implements Resolver.
func (r *DNSResolver) Resolve(ctx context.Context, name string) (path.Path, error) {
	return r.ResolveN(ctx, name, DefaultDepthLimit)
}

// ResolveN implements Resolver.
func (r *DNSResolver) ResolveN(ctx context.Context, name string, depth int) (path.Path, error) {
	return Resolve(ctx, r, name, depth)
}

type lookupRes struct {
	path  string
	error error
}

// ResolveOnce implements Lookup.
// TXT records for a given domain name should contain a b58
// encoded multihash.
func (r *DNSResolver) ResolveOnce(ctx context.Context, name string) (string, error) {
	name = removePathPrefix(name)
	segments := strings.SplitN(name, "/", 2)
	domain := segments[0]

	if !isd.IsDomain(domain) {
		return "", fmt.Errorf("Not a valid domain name: [%s]", domain)
	}
	log.Debugf("DNSResolver resolving %s", domain)

	rootChan := make(chan lookupRes, 1)
	go workDomain(r, domain, rootChan)

	subChan := make(chan lookupRes, 1)
	go workDomain(r, "_dnslink."+domain, subChan)

	var subRes lookupRes
	select {
	case subRes = <-subChan:
	case <-ctx.Done():
		return "", ctx.Err()
	}

	var p string
	if subRes.error == nil {
		p = subRes.path
	} else {
		var rootRes lookupRes
		select {
		case rootRes = <-rootChan:
		case <-ctx.Done():
			return "", ctx.Err()
		}
		if rootRes.error == nil {
			p = rootRes.path
		} else {
			return "", ErrResolveFailed
		}
	}
	if len(segments) > 1 {
		return strings.TrimRight(p, "/") + "/" + segments[1], nil
	}
	return p, nil
}

func workDomain(r *DNSResolver, name string, res chan lookupRes) {
	txt, err := r.lookupTXT(name)
	log.Debugf("DNSResolver lookupTXT(%s) => %s", name, txt)

	if err != nil {
		// Error is != nil
		res <- lookupRes{"", err}
		return
	}

	for _, t := range txt {
		p, err := parseEntry(t)
		if err == nil {
			res <- lookupRes{p, nil}
			return
		}
		log.Debugf("Could not parse entry %s", t)
	}
	res <- lookupRes{"", ErrResolveFailed}
}

func parseEntry(txt string) (string, error) {
	p, err := path.ParseCidToPath(txt) // bare IPFS multihashes
	if err == nil {
		return p.String(), nil
	}

	if isIprsPath(txt) {
		return txt, nil
	}

	return tryParseDnsLink(txt)
}

// Parse ipfs/ipns/iprs links of the form
// dnslink=/ipfs/somepath
func tryParseDnsLink(txt string) (string, error) {
	parts := strings.SplitN(txt, "=", 2)
	if len(parts) == 2 && parts[0] == "dnslink" {
		// Check if it's an ipfs or ipns path
		p, err := path.ParsePath(parts[1])
		if err == nil {
			return p.String(), nil
		}

		// Check if it's an iprs path
		if isIprsPath(parts[1]) {
			return parts[1], nil
		}
	}

	return "", errors.New("Not a valid dnslink entry")
}

// Must be of the form
// /iprs/<hash>/somepath
// /iprs/www.example.com/somepath
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
	if parts[2] == "" {
		return false
	}
	if parts[2] == "" {
		return false
	}
	return u.IsValidHash(parts[2]) || isd.IsDomain(parts[2])
}
