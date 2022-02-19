package expert

import (
	"errors"
	"log"
	"os"
)

// const
const (
	thresholdMin = 23
	thresholdMax = 40

	minSize = 1
	maxSize = 180

	logPrefix = "ea:\t"
)

// var
var (
	activeTrades     map[Pair]*TradeParams
	invalidSizeError = errors.New("invalid size argument")
	//invalidSizeActionError = errors.New("invalid size for action argument")
)

// structs
type Candle struct {
	Pair Pair

	High   float64
	Low    float64
	Open   float64
	Close  float64
	Volume float64

	// Holds other information like RSI, MA or any other data the system needs
	OtherData map[string]float64

	Time   int64
	Closed bool
}

// Struct for initiating a trade
type TradeParams struct {
	OpenTradeAt  float64 `json:"open_trade_at"`
	TakeProfitAt float64 `json:"take_profit_at"`
	StopLossAt   float64 `json:"stop_loss_at"`
	TradeSize    string  `json:"trade_size"`
	Rating       int     `json:"rating"`
	Pair         Pair    `json:"pair"`
}

type SellParams struct {
	IsStopLoss  bool
	SellTradeAt float64
	PL          float64
	Pair        Pair
}

type Config struct {
	Size            int
	BuyAction       BuyAction
	SellAction      SellAction
	Storage         DataSource
	DefaultAnalysis []*CalculateAction
}

// interface to save and retrieve candles
type DataSource interface {
	FetchCandles(pair Pair, size int) ([]*Candle, error)
	Persist(candle *Candle) error
}

// function for analyze the data set, returns a %value, if the trade is worth taking
type Transform func([]*Candle) *TradeParams

type BuyAction func(*TradeParams) bool
type SellAction func(*SellParams) bool

type Pair string

type CalculateAction struct {
	Name   string
	Size   int8
	Action func([]*Candle) float64
}

type system struct {
	size            int
	datasource      DataSource
	buyAction       BuyAction
	sellAction      SellAction
	log             *log.Logger
	calculateAction []*CalculateAction
}

type Trader interface {
	Record(candle *Candle, transform Transform, tradeSize string)
	TradeClosed(pair Pair)
	OnError(error)
}

// init
func init() {
	activeTrades = make(map[Pair]*TradeParams)
}

// methods
func (c *Candle) IsUp() bool {
	return c.Close > c.Open
}

func (s *system) Record(candle *Candle, transform Transform, tradeSize string) {
	if !candle.Closed {
		//Still actively traded
		s.tryClosing(candle)
		return
	}

	// apply actions
	for _, action := range s.calculateAction {
		candles, err := s.datasource.FetchCandles(candle.Pair, int(action.Size-1))
		if err != nil {
			s.log.Printf("Unable to apply calculate:%+v", err)
			break
		}

		if len(candles) != int(action.Size-1) {
			//s.log.Printf("Found len:%d, expected:%d\n", len(candles)+1, action.Size)
			break
		}

		candle.OtherData[action.Name] = action.Action(append([]*Candle{candle}, candles...))
	}

	// persist the new candle
	if err := s.datasource.Persist(candle); err != nil {
		s.log.Println(err)
		return
	}

	if _, ok := activeTrades[candle.Pair]; ok {
		// we currently have an active trade
		s.tryClosing(candle)
		return
	}

	dataset, err := s.datasource.FetchCandles(candle.Pair, s.size)
	if err != nil {
		s.log.Println(err)
		return
	}

	if len(dataset) != s.size {
		//s.log.Printf("invalid dataset size:%d expected:%d for:%s\n", len(dataset), s.size, candle.Pair)
		return
	}

	result := transform(dataset)
	result.TradeSize = tradeSize

	// TODO: Use machine learning.
	if result.Rating > thresholdMin && result.Rating < thresholdMax {
		// Call next function
		if s.buyAction(result) {
			activeTrades[candle.Pair] = result
		}
	}
}

func (s *system) TradeClosed(pair Pair) {
	delete(activeTrades, pair)
}

func (s *system) OnError(err error) {
	s.log.Printf("An error occurred: %+v\n", err)
}

func (s *system) tryClosing(candle *Candle) {
	params, ok := activeTrades[candle.Pair]
	if !ok {
		// we currently have an active trade
		return
	}

	if candle.Close <= params.StopLossAt {
		if s.sellAction(&SellParams{
			IsStopLoss:  true,
			SellTradeAt: params.TakeProfitAt,
			PL:          params.StopLossAt - params.OpenTradeAt,
			Pair:        candle.Pair,
		}) {
			s.TradeClosed(candle.Pair)
		}
	} else if candle.Close >= params.TakeProfitAt {
		if s.sellAction(&SellParams{
			IsStopLoss:  false,
			SellTradeAt: params.TakeProfitAt,
			PL:          params.TakeProfitAt - params.OpenTradeAt,
			Pair:        candle.Pair,
		}) {
			s.TradeClosed(candle.Pair)
		}
	}
}

// NewTrader returns a new Trader with logger enabled
func NewTrader(config *Config) Trader {
	return NewTraderWithLogger(config, log.New(os.Stdout, logPrefix, log.LstdFlags|log.Lshortfile))
}

func NewTraderWithLogger(config *Config, logger *log.Logger) Trader {
	return &system{
		size:            config.Size,
		datasource:      config.Storage,
		buyAction:       config.BuyAction,
		sellAction:      config.SellAction,
		log:             logger,
		calculateAction: config.DefaultAnalysis,
	}
}
