package recordstore_types

import (
	"bytes"
	"fmt"
	pb "github.com/dirkmc/go-iprs/pb"
	proto "github.com/gogo/protobuf/proto"
)

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
