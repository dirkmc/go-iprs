package iprs_path

import (
	"strings"
	"testing"

	u "gx/ipfs/QmPsAfmDBnZN3kZGSuNwvCNDZiHneERSKmRcFyG3UkvcT3/go-ipfs-util"
	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
)

func TestTargetParsing(t *testing.T) {
	ipfsPath := "/ipfs/QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN"
	ipfsPathCid, err := cid.Parse(ipfsPath)
	if err != nil {
		t.Fatal(err)
	}
	v0c := cid.NewCidV0(u.Hash([]byte("value")))
	v1c := cid.NewCidV1(cid.GitRaw, u.Hash([]byte("value")))

	type expectation struct {
		c   *cid.Cid
		p   string
		err bool
	}
	type testCase struct {
		sample []byte
		exp    expectation
	}

	cases := []testCase{
		// Raw CIDv0 bytes
		testCase{
			sample: v0c.Bytes(),
			exp:    expectation{v0c, "", false},
		},
		// CIDv0 string
		testCase{
			sample: []byte(v0c.String()),
			exp:    expectation{v0c, "", false},
		},
		// CIDv0 string with path
		testCase{
			sample: []byte(v0c.String() + "/test/path"),
			exp:    expectation{v0c, "/test/path", false},
		},
		// Raw CIDv1 bytes
		testCase{
			sample: v1c.Bytes(),
			exp:    expectation{v1c, "", false},
		},
		// CIDv1 string
		testCase{
			sample: []byte(v1c.String()),
			exp:    expectation{v1c, "", false},
		},
		// CIDv1 string with path
		testCase{
			sample: []byte(v1c.String() + "/test/path"),
			exp:    expectation{v1c, "/test/path", false},
		},
		// IPFS path string
		testCase{
			sample: []byte(ipfsPath),
			exp:    expectation{ipfsPathCid, "", false},
		},
		// IPFS path string with path
		testCase{
			sample: []byte(ipfsPath + "/test/path"),
			exp:    expectation{ipfsPathCid, "/test/path", false},
		},
		// Old style IPNS raw multihash bytes
		testCase{
			sample: []byte(v0c.Hash()),
			exp:    expectation{v0c, "", false},
		},
		// Fail to parse ipns paths
		testCase{
			sample: []byte("/ipns/QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN"),
			exp:    expectation{v0c, "", true},
		},
		// Fail to parse iprs paths
		testCase{
			sample: []byte("/iprs/QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN/id"),
			exp:    expectation{v0c, "", true},
		},
		// Fail to parse random garbage
		testCase{
			sample: []byte("blahblah"),
			exp:    expectation{v0c, "", true},
		},
	}

	for _, tcase := range cases {
		c, p, err := ParseTargetToCid(tcase.sample)
		if tcase.exp.err {
			if err == nil {
				t.Fatalf("expected error parsing %s", tcase.sample)
			}
			break
		}
		if !tcase.exp.err && err != nil {
			t.Fatalf("error parsing %s: %s", tcase.sample, err)
		}
		if !c.Equals(tcase.exp.c) {
			t.Fatalf("cid mismatch parsing %s:\ngot      %s\nexpected %s", tcase.sample, c, tcase.exp.c)
		}
		pstr := ""
		if len(p) > 0 {
			pstr = "/" + strings.Join(p, "/")
		}
		if pstr != tcase.exp.p {
			t.Fatalf("path mismatch parsing %s:\ngot      %s\nexpected %s", tcase.sample, pstr, tcase.exp.p)
		}
	}
}
