package iprs_resolver

import (
	"context"
	"fmt"
	"net"
	"strings"

	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
	isd "gx/ipfs/QmZmmuAXgX73UQmX1jRKjTGmjzq24Jinqkq8vzkBtno4uX/go-is-domain"
	path "github.com/ipfs/go-ipfs/path"
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
	return &DNSResolver{net.LookupTXT}
}
/*
// newDNSResolver constructs a name resolver using DNS TXT records,
// returning a Lookup instead of NewDNSResolver's Resolver.
func newDNSResolver() Lookup {
	return &DNSResolver{net.LookupTXT}
}*/

type lookupRes struct {
	path  string
	error error
}

func (r *DNSResolver) Resolve(ctx context.Context, domain string) (string, error) {
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

	if subRes.error == nil {
		return subRes.path, nil
	}

	var rootRes lookupRes
	select {
	case rootRes = <-rootChan:
	case <-ctx.Done():
		return "", ctx.Err()
	}
	if rootRes.error == nil {
		return rootRes.path, nil
	}

	return "", ErrResolveFailed
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

// Parse links of the form
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

	return "", fmt.Errorf("Not a valid dnslink entry: %s", txt)
}

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
	if parts[2] == "" {
		return false
	}
	if parts[2] == "" {
		return false
	}
	if isd.IsDomain(parts[2]) {
		return true
	}
	_, err := cid.Parse(parts[2])
	return err != nil
}
