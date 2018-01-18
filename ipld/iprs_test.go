package iprs_ipld

import (
	"bytes"
	"sort"
	"testing"

	node "gx/ipfs/QmNwUEK7QbwSqyKBu3mMtToo8SUc6wQJ7gdZq4gGGJqfnf/go-ipld-format"
	u "gx/ipfs/QmPsAfmDBnZN3kZGSuNwvCNDZiHneERSKmRcFyG3UkvcT3/go-ipfs-util"
	blocks "gx/ipfs/QmYsEQydGrsxNZfAiskvQ76N2xE9hDQtSAkRSynwMiUK3c/go-block-format"
	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
)

func TestMarshalIprsNodeRoundtrip(t *testing.T) {
	valueCid := cid.NewCidV0(u.Hash([]byte("value")))
	vft := VerificationType_Key
	verification := map[string]*cid.Cid{"mycid": cid.NewCidV0(u.Hash([]byte("value")))}
	vlt := ValidationType_EOL
	validation := []byte("validation")
	validity := &Validity{
		VerificationType: vft,
		Verification:     verification,
		ValidationType:   vlt,
		Validation:       validation,
	}
	signature := []byte("sig")

	// 1. Newly constructed Node
	o, err := NewIprsNode(valueCid.Bytes(), validity, signature)
	if err != nil {
		t.Fatal(err)
	}

	// 2. Decoded newly constructed Node
	nd, err := DecodeIprsBlock(o)
	if err != nil {
		t.Fatal(err)
	}

	// 3. Decoded block
	b, err := blocks.NewBlockWithCid(nd.RawData(), nd.Cid())
	if err != nil {
		t.Fatal(err)
	}
	nb, err := DecodeIprsBlock(b)
	if err != nil {
		t.Fatal(err)
	}

	var tests = func(n node.Node) {
		if n.Cid().Type() != CodecIprsCbor {
			t.Fatalf("node CID is of Type %d, expected %d", n.Cid().Type(), CodecIprsCbor)
		}

		// Link in Verification data added above
		if len(n.Links()) != 1 {
			t.Fatalf("have %d links, expected %d", len(n.Links()), 1)
		}

		vali, _, err := n.Resolve([]string{"value"})
		val, ok := vali.([]byte)
		if err != nil || !ok {
			t.Fatal("incorrectly formatted value")
		}
		if bytes.Compare(val, valueCid.Bytes()) != 0 {
			t.Fatalf("value is %s, expected %s", val, valueCid)
		}

		versioni, _, err := n.Resolve([]string{"version"})
		versionr, ok := versioni.(uint64)
		if err != nil || !ok {
			t.Fatal("incorrectly formatted version")
		}
		if IprsVerificationType(versionr) != 1 {
			t.Fatalf("version is %d, expected %d", versionr, 1)
		}

		vfti, _, err := n.Resolve([]string{"validity", "verificationType"})
		vftr, ok := vfti.(uint64)
		if err != nil || !ok {
			t.Fatal("incorrectly formatted verificationType")
		}
		if IprsVerificationType(vftr) != vft {
			t.Fatalf("verificationType is %d, expected %d", vftr, vft)
		}

		verificationi, _, err := n.Resolve([]string{"validity", "verification"})
		vfn, ok := verificationi.(map[string]interface{})
		if err != nil || !ok {
			t.Fatalf("incorrectly formatted verification %T", vfn)
		}
		if !vfn["mycid"].(*cid.Cid).Equals(verification["mycid"]) {
			t.Fatalf("verification is %s, expected %s", vfn, verification)
		}

		vlti, _, err := n.Resolve([]string{"validity", "validationType"})
		vltr, ok := vlti.(uint64)
		if err != nil || !ok {
			t.Fatal("incorrectly formatted validationType")
		}
		if IprsValidationType(vltr) != vlt {
			t.Fatalf("validationType is %d, expected %d", vltr, vlt)
		}

		validationi, _, err := n.Resolve([]string{"validity", "validation"})
		vld, ok := validationi.([]byte)
		if err != nil || !ok {
			t.Fatal("incorrectly formatted validation")
		}
		if bytes.Compare(vld, validation) != 0 {
			t.Fatalf("validation is %s, expected %s", vld, validation)
		}

		sigi, _, err := n.Resolve([]string{"signature"})
		sig, ok := sigi.([]byte)
		if err != nil || !ok {
			t.Fatal("incorrectly formatted signature")
		}
		if bytes.Compare(sig, signature) != 0 {
			t.Fatalf("signature is %s, expected %s", sig, signature)
		}

		full := []string{
			"version",
			"value",
			"validity",
			"validity/verificationType",
			"validity/verification",
			"validity/verification/mycid",
			"validity/validationType",
			"validity/validation",
			"signature",
		}

		top := []string{
			"version",
			"value",
			"validity",
			"signature",
		}

		validityFull := []string{
			"verificationType",
			"verification",
			"verification/mycid",
			"validationType",
			"validation",
		}

		validityTop := []string{
			"verificationType",
			"verification",
			"validationType",
			"validation",
		}

		assertStringsEqual(t, full, n.Tree("", -1))
		assertStringsEqual(t, []string{}, n.Tree("", 0))
		assertStringsEqual(t, top, n.Tree("", 1))
		assertStringsEqual(t, validityFull, n.Tree("validity", -1))
		assertStringsEqual(t, validityTop, n.Tree("validity", 1))
	}

	tests(o)
	tests(nb)
	tests(nd)
	tests(nd.Copy())
}

func assertStringsEqual(t *testing.T, a, b []string) {
	sort.Strings(a)
	sort.Strings(b)

	if len(a) != len(b) {
		t.Fatalf("lengths differed:\n%s\n%s\n", a, b)
	}

	for i, v := range a {
		if v != b[i] {
			t.Fatalf("got mismatch:\n%s\n%s\n", a, b)
		}
	}
}
