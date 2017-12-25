package iprs_resolver

import (
	"errors"
	"context"
	"strings"

	path "github.com/ipfs/go-ipfs/path"
	proquint "gx/ipfs/QmYnf27kzqR2cxt6LFZdrAFJuQd6785fTkBvMuEj9EeRxM/proquint"
)

type ProquintResolver struct{}

// Resolve implements Resolver.
func (r *ProquintResolver) Resolve(ctx context.Context, name string) (path.Path, error) {
	return r.ResolveN(ctx, name, DefaultDepthLimit)
}

// ResolveN implements Resolver.
func (r *ProquintResolver) ResolveN(ctx context.Context, name string, depth int) (path.Path, error) {
	return Resolve(ctx, r, name, depth)
}

// resolveOnce implements resolver. Decodes the proquint string.
func (r *ProquintResolver) ResolveOnce(ctx context.Context, name string) (string, error) {
	name = removePathPrefix(name)
	segments := strings.SplitN(name, "/", 2)

	ok, err := proquint.IsProquint(segments[0])
	if err != nil || !ok {
		return "", errors.New("not a valid proquint string")
	}
	
	decoded := string(proquint.Decode(segments[0]))
	if len(segments) > 1 {
		return strings.TrimRight(name, "/") + "/" + segments[1], nil
	}
	
	return decoded, nil
}
