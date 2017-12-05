package recordstore_types

import (
	"fmt"
	"math/rand"
	pb "github.com/dirkmc/go-iprs/pb"
	proto "github.com/gogo/protobuf/proto"
)

type selectFunction func([]*pb.IprsEntry, [][]byte) (int, error)

func shuffle(a []*pb.IprsEntry) {
	for n := 0; n < 5; n++ {
		for i, _ := range a {
			j := rand.Intn(len(a))
			a[i], a[j] = a[j], a[i]
		}
	}
}

func AssertSelected(selectFn selectFunction, r *pb.IprsEntry, from []*pb.IprsEntry) error {
	shuffle(from)
	var vals [][]byte
	for _, r := range from {
		data, err := proto.Marshal(r)
		if err != nil {
			return err
		}
		vals = append(vals, data)
	}

	i, err := selectFn(from, vals)
	if err != nil {
		return err
	}

	if from[i] != r {
		return fmt.Errorf("selected incorrect record %d", i)
	}

	return nil
}
