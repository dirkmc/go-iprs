package main

import (
	pb "github.com/libp2p/go-libp2p-record/pb"
)

type MemRecordStore struct {
	cache map[string]*pb.Record
	keyToHash map[string]string
}

func NewRecordStore() (*MemRecordStore, error) {
	rs := &MemRecordStore{
		cache: make(map[string]*pb.Record),
		keyToHash: make(map[string]string),
	}
	return rs, nil
}

func (rs *MemRecordStore) Get(key string) ([]pb.Record, error) {
	h := rs.keyToHash[key]
	r := rs.cache[h]
	if r == nil {
		return []pb.Record{}, nil
	}
	return []pb.Record{*r}, nil
}

func (rs *MemRecordStore) Put(key string, recordSigMultihash []byte) error {
	rs.keyToHash[key] = string(recordSigMultihash)
	return nil
}

func (rs *MemRecordStore) CacheStore(recordSigMultihash []byte, obj *pb.Record) error {
	rs.cache[string(recordSigMultihash)] = obj
	return nil
}

