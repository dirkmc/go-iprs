package main

import (
	pb "github.com/libp2p/go-libp2p-record/pb"
)

type KadRecordStore struct {}
/*
func NewRecordStore() (*KadRecordStore, error) {
	return &KadRecordStore{}, nil
}
*/
func (r *KadRecordStore) Get(key string) ([]pb.Record, error) {
	return []pb.Record{}, nil
}

func (r *KadRecordStore) Put(key string, recordSigMultihash []byte) error {
	return nil
}
