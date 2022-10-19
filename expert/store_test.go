package expert

import (
	"errors"
	"fmt"
	"testing"

	"github.com/oblessing/artisgo/bot/store"
	"sync"
)

func TestExpertMapping(t *testing.T) {
	tmpStorage := &tmpStorage{}
	str := NewDataSource(tmpStorage)
	tmpStorage.cleanup()

	sCandle := &Candle{
		Pair:   "test",
		High:   1,
		Low:    2,
		Open:   3,
		Close:  4,
		Volume: 5,
		Time:   6,
		Closed: true,
	}

	if err := str.Persist(sCandle); err != nil {
		t.Fatalf("Error persitin data in store: %+v", err)
	}

	bCandle := tmpStorage.store["test"]

	if bCandle[0].IsClosed != sCandle.Closed {
		t.Fatalf("IsClosed is invalid")
	}

	if bCandle[0].Candle.Low != sCandle.Low {
		t.Fatalf("Low is invalid")
	}

	if bCandle[0].Candle.High != sCandle.High {
		t.Fatalf("High is invalid")
	}

	if bCandle[0].Candle.Open != sCandle.Open {
		t.Fatalf("Open is invalid")
	}

	if bCandle[0].Candle.Close != sCandle.Close {
		t.Fatalf("Close is invalid")
	}

	if bCandle[0].Candle.Vol != sCandle.Volume {
		t.Fatalf("Vol is invalid")
	}

	if bCandle[0].Pair != string(sCandle.Pair) {
		t.Fatalf("Pair is invalid")
	}
}
