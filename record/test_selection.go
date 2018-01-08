package iprs_record

import (
	"fmt"
	"math/rand"
)

type selectFunction func([]*Record) (int, error)

func shuffle(a []*Record) {
	for n := 0; n < 5; n++ {
		for i, _ := range a {
			j := rand.Intn(len(a))
			a[i], a[j] = a[j], a[i]
		}
	}
}

func AssertSelected(selectFn selectFunction, expected *Record, from []*Record) error {
	shuffle(from)
	i, err := selectFn(from)
	if err != nil {
		return err
	}

	if from[i] != expected {
		return fmt.Errorf("selected incorrect record %d", i)
	}

	return nil
}
