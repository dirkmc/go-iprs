package recordstore_record

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"
	logging "github.com/ipfs/go-log"
	pb "github.com/dirkmc/go-iprs/pb"
	proto "github.com/gogo/protobuf/proto"
	routing "gx/ipfs/QmPR2JzfKd9poHx9XBhzoFeBBC31ZM3W5iUPKJZWyaoZZm/go-libp2p-routing"
)

const PublishEntryTimeout = time.Second*10

// ErrExpiredRecord should be returned when an Iprs record is
// invalid due to being too old
var ErrExpiredRecord = errors.New("expired record")

var log = logging.Logger("recordstore.record")

func RecordDataForSig(r *pb.IprsEntry) []byte {
	return bytes.Join([][]byte{
		r.Value,
		r.Validity,
		[]byte(fmt.Sprint(r.GetValidityType())),
	},
		[]byte{})
}

func UnmarshalRecords(vals [][]byte) []*pb.IprsEntry {
	var recs []*pb.IprsEntry
	for _, v := range vals {
		e := new(pb.IprsEntry)
		err := proto.Unmarshal(v, e)
		if err == nil {
			recs = append(recs, e)
		} else {
			recs = append(recs, nil)
		}
	}

	return recs
}

func PutEntryToRouting(ctx context.Context, r routing.ValueStore, iprsKey string, entry *pb.IprsEntry) error {
	data, err := proto.Marshal(entry)
	if err != nil {
		return err
	}

	timectx, cancel := context.WithTimeout(ctx, PublishEntryTimeout)
	defer cancel()

	log.Debugf("Storing iprs entry at %s", iprsKey)
	return r.PutValue(timectx, iprsKey, data)
}
