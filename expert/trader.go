package expert

import (
	"errors"
	"log"
	"os"
)

// const
const (
	thresholdMin = 20
	thresholdMax = 40

	minSize = 1
	maxSize = 180

	logPrefix = "ea:\t"
)

// var
var (
	activeTrades     map[Pair]bool
	invalidSizeError = errors.New("invalid size argument")
)

// structs
type Candle struct {
	Pair Pair

	High   float64
	Low    float64
	Open   float64
	Close  float64
	Volume float64

	Time   float64
	Closed bool
}

// Struct for initiating a trade
type TradeParams struct {
	OpenTradeAt  float64
	TakeProfitAt float64
	StopLossAt   float64
	Rating       int
	Pair         Pair
}

// interface to save and retrieve candles
type DataSource interface {
	FetchCandles(pair Pair, size int) ([]*Candle, error)
	Persist(candle *Candle) error
}

// function for analyze the data set, returns a %value, if the trade is worth taking
type Transform func([]*Candle) *TradeParams

type Action func(*TradeParams) bool

type Pair string

type system struct {
	size       int
	datasource DataSource
	action     Action
	log        *log.Logger
}

type Trader interface {
	Record(candle *Candle, transform Transform)
	TradeClosed(pair Pair)
}

// init
func init() {
	activeTrades = map[Pair]bool{}
}

// methods
func (s *system) Record(candle *Candle, transform Transform) {
	if !candle.Closed {
		//Still actively traded
		return
	}
	// persist the new candle
	if err := s.datasource.Persist(candle); err != nil {
		s.log.Println(err)
		return
	}

	if _, ok := activeTrades[candle.Pair]; ok {
		// we currently have an active trade
		return
	}

	dataset, err := s.datasource.FetchCandles(candle.Pair, s.size)
	if err != nil {
		s.log.Println(err)
		return
	}

	if len(dataset) != s.size {
		s.log.Printf("invalid dataset size:%d expected:%d", len(dataset), s.size)
		return
	}

	result := transform(dataset)
	if result.Rating > thresholdMin && result.Rating < thresholdMax {
		// Call next function
		activeTrades[candle.Pair] = s.action(result)
	}
}

func (s *system) TradeClosed(pair Pair) {
	delete(activeTrades, pair)
}

// export
func NewTrader(size int, action Action, storage DataSource) (Trader, error) {
	return NewTraderWithLogger(size, action, storage, log.New(os.Stdout, logPrefix, log.LstdFlags|log.Lshortfile))
}

func NewTraderWithLogger(size int, action Action, storage DataSource, logger *log.Logger) (Trader, error) {
	if size < minSize || size > maxSize {
		return nil, invalidSizeError
	}

	return &system{
		size:       size,
		datasource: storage,
		action:     action,
		log:        logger,
	}, nil
}
