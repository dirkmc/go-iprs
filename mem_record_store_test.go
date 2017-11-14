package recordstore

import (
	"fmt"
	"log"
	"testing"
	"bytes"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"crypto/rand"
	//gologging "github.com/whyrusleeping/go-logging"
	//ipld "github.com/ipfs/go-ipld-cbor"
	//mh "github.com/multiformats/go-multihash"
	record "github.com/libp2p/go-libp2p-record"
)

func TestGetPut(t *testing.T) {
	sk, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	if err != nil {
		log.Fatal(err)
	}

	rs, err := NewRecordStore()
	if err != nil {
		log.Fatal(err)
	}

	r, nil := record.MakePutRecord(sk, "myrec", []byte("myval"), true)
	if err != nil {
		log.Fatal(err)
	}

	// TODO: Store IPLD sig hash rather than the record sig hash?
	recordSigMultihash := r.GetSignature()
	err = rs.CacheStore(recordSigMultihash, r)
	if err != nil {
		log.Fatal(err)
	}

	err = rs.Put("bananas", recordSigMultihash)
	if err != nil {
		log.Fatal(err)
	}

	records, err := rs.Get("bananas")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Records %v\n", records)
	if !bytes.Equal(records[0].GetValue(), r.GetValue()) || records[0].GetKey() != r.GetKey() {
		t.Error("Record that was put doesn't match record that was retrieved")
	}
}
