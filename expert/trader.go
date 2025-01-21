package expert

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	settings "github.com/oblessing/artisgo"
	"github.com/oblessing/artisgo/logger"
	"github.com/oblessing/artisgo/store"
)

const (
	TradeTypeLong  TradeType = "long"
	TradeTypeShort TradeType = "short"
)

var (
	// TODO: Add support for placing multiple trades for a specific symbol
	activeTrades = sync.Map{} // map[Pair]*TradeParams{}

	nextReset = time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day()+1, 0, 0, 0, 0, time.UTC)
)

type TradeType string

// Transform for analyze the data set, returns a %value, if the trade is worth taking
type Transform func(ctx context.Context, trigger Candle, candles []*Candle) *TradeParams

type Pair string

type Candle struct {
	Pair   Pair
	High   float64
	Low    float64
	Open   float64
	Close  float64
	Volume float64
	// Holds other information like RSI, MA or any other data the system needs
	OtherData map[string]float64
	Time      int64
	Closed    bool
}

func (c *Candle) IsUp() bool {
	return c.Close > c.Open
}

// TradeParams for initiating a trade
type TradeParams struct {
	TradeType         TradeType          `json:"trade_type"`
	OriginalTradeType TradeType          `json:"original_trade_type"`
	OpenTradeAt       string             `json:"open_trade_at"`
	Volume            float64            `json:"volume"`
	OrderID           string             `json:"order_id"`
	TakeProfitAt      string             `json:"take_profit_at"`
	StopLossAt        string             `json:"stop_loss_at"`
	TradeSize         string             `json:"trade_size"`
	Rating            int                `json:"rating"` // Deprecated
	Pair              Pair               `json:"pair"`
	CreatedAt         time.Time          `json:"time"`
	Attribs           map[string]float64 `json:"others"`
	CanNotOverride    bool               `json:"canNotOverride"`
	AutomaticClose    bool               `json:"automaticClose"`
}

func (t TradeParams) OpenTradeAtV() float64 {
	r, _ := strconv.ParseFloat(t.OpenTradeAt, 64)
	return r
}
func (t TradeParams) TakeProfitAtV() float64 {
	r, _ := strconv.ParseFloat(t.TakeProfitAt, 64)
	return r
}
func (t TradeParams) StopLossAtV() float64 {
	r, _ := strconv.ParseFloat(t.StopLossAt, 64)
	return r
}

type TradeData struct {
	OrderID       string
	ClientOrderID string
}

type SellParams struct {
	IsStopLoss  bool
	SellTradeAt float64
	PL          float64
	OrderID     string
	TradeSize   string
	Pair        Pair
	TradeType   TradeType `json:"trade_type"`
}

type CalculateAction struct {
	Name   string
	Size   int8 // This is ignored
	Action func([]*Candle) float64
}

type system struct {
	settings     settings.Config
	datasource   DataSource
	orderService OrderService
	// make it a map if we plan to support multiple positions
	rw sync.RWMutex
}

type RecordConfig struct {
	// Represents the percentage change
	LotSize         float64
	RatioToOne      float64
	CandleSize      int
	AdditionalData  []string // minPrice, stepSize, precision
	DefaultAnalysis []*CalculateAction
}

type DataSource interface {
	FetchCandles(ctx context.Context, pair Pair, size int) ([]*Candle, error)
	Persist(ctx context.Context, candle *Candle) error
}

type OrderService interface {
	PlaceTrade(ctx context.Context, params TradeParams) (TradeData, error)
	CloseTrade(ctx context.Context, params SellParams) (bool, error)
}

type Trader interface {
	Record(ctx context.Context, candle *Candle, transform Transform, config RecordConfig)
}

func NewExpertTrader(config settings.Config, storage store.Database, service OrderService) *system {
	return &system{
		settings:     config,
		datasource:   NewDataSource(storage),
		orderService: service,
	}
}

func (s *system) Record(ctx context.Context, c *Candle, transform Transform, config RecordConfig) {
	// // Try checking if we need to close any trade,
	// // do not use heikin ashi to close trade.
	s.tryClosing(ctx, c)

	// Check if the time is still running,
	// return, so we do not persist it.
	if !c.Closed {
		return
	}

	// Convert card to heikin ashi.
	candles, _ := s.datasource.FetchCandles(ctx, c.Pair, config.CandleSize)
	var previousCandle = c
	if len(candles) != 0 {
		previousCandle = candles[len(candles)-1]
	}
	candle := convertToHeikinAshi(previousCandle, c)

	// apply actions; MA, RSI, etc
	for _, action := range config.DefaultAnalysis {
		if action.Name == "LASTCLOSE" {
			candle.OtherData[action.Name] = action.Action(append([]*Candle{c}))

			continue
		}

		d := append([]*Candle{}, candles...)
		d = append(d, candle)
		if time.Now().After(nextReset) {
			t := time.Now()
			// we should reset our record
			nextReset = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, time.UTC)
			for _, v := range d {
				// we should reset any 24 hour indicator
				delete(v.OtherData, "LL24")
				delete(v.OtherData, "HH24")
			}
		}

		candle.OtherData[action.Name] = action.Action(d)
	}

	// persist the new candle
	if err := s.datasource.Persist(ctx, candle); err != nil {
		logger.Error(ctx, "error persisting record", zap.Error(err))
		return
	}

	if len(candles) < 2 {
		return
	}

	s.processTrade(ctx, *c, transform, config, candles)
}

func (s *system) processTrade(ctx context.Context, c Candle, transform Transform, config RecordConfig, dataset []*Candle) {
	lotPrecision := findNumberOfDecimal(config.AdditionalData[1])
	quotePrecision := findNumberOfDecimal(config.AdditionalData[0])

	if len(dataset) == 1 {
		return
	}

	result := transform(ctx, c, dataset)
	// Check if we can act on the data
	if result == nil {
		return
	}

	// lets try delayed data
	prevCandleAnalysis := dataset[len(dataset)-1].OtherData

	// trading amount is
	var tradeSize = ((1 / result.OpenTradeAtV()) * s.settings.TradeAmount) * config.LotSize
	var buyPrice = fmt.Sprintf("%v", RoundToDecimalPoint(result.OpenTradeAtV(), quotePrecision))

	tr, _ := prevCandleAnalysis["TR"]
	atr, _ := prevCandleAnalysis["ATR"]
	ma, _ := prevCandleAnalysis["MA"]
	hh24h, _ := prevCandleAnalysis["HH24"]
	var (
		tp float64
		ot float64
	)

	switch result.TradeType {
	case TradeTypeLong:
		stopLoss := result.OpenTradeAtV() - ((result.OpenTradeAtV()) / config.LotSize)
		// since leverage is 10 times
		// current price + ((current price * ratio) / 10)
		var takeProfit = result.OpenTradeAtV() + ((result.OpenTradeAtV() * config.RatioToOne) / config.LotSize)
		result.TakeProfitAt = fmt.Sprintf("%v", RoundToDecimalPoint(takeProfit, quotePrecision))
		result.StopLossAt = fmt.Sprintf("%v", RoundToDecimalPoint(stopLoss, quotePrecision))
		result.TradeSize = fmt.Sprintf("%v", RoundToDecimalPoint(tradeSize, lotPrecision))
		tp = takeProfit
		ot = result.OpenTradeAtV()
	case TradeTypeShort:
		stopLoss := result.OpenTradeAtV() + ((result.OpenTradeAtV()) / config.LotSize)
		// since leverage is 10 times
		// current price + ((current price * ratio) / 10)
		var takeProfit = result.OpenTradeAtV() - ((result.OpenTradeAtV() * config.RatioToOne) / config.LotSize)
		result.TakeProfitAt = fmt.Sprintf("%v", RoundToDecimalPoint(takeProfit, quotePrecision))
		result.StopLossAt = fmt.Sprintf("%v", RoundToDecimalPoint(stopLoss, quotePrecision))
		result.TradeSize = fmt.Sprintf("%v", RoundToDecimalPoint(tradeSize, lotPrecision))
		tp = takeProfit
		ot = result.OpenTradeAtV()
	}
	// set timestamp
	result.CreatedAt = time.Now().UTC()
	// Set additional attribs for logging //  digit rsi -> short -> down stops at (6), 83 + xtreme
	result.Attribs = prevCandleAnalysis
	result.OpenTradeAt = buyPrice
	result.Volume = c.Volume

	skipa := false
	skipb := false
	skipc := false

	if tr < (atr * 1.3) {
		// we should enforce this.
		// logger.Warn(ctx, "atr is less than 1.3x", zap.Any("result", result))

		// we need momentum
		// TODO: We might remove it.
		skipa = true
	}

	// TODO: Check if TP is above or below MA (short TP should be above MA) invert for long
	if !((result.TradeType == TradeTypeShort && tp > ma) || (result.TradeType == TradeTypeLong && tp < ma)) {
		skipb = true
	}

	change := ((hh24h - c.Close) / ((hh24h + c.Close) / 2)) * 100

	// TODO: Check if OT is above or below MA (short OT should be above MA) invert for long
	// look at this
	if !((result.TradeType == TradeTypeShort && ot > ma) || (result.TradeType == TradeTypeLong && ot < ma)) {
		skipc = true
	}

	logger.Warn(ctx, "trade info", zap.Any("skipa", skipa), zap.Any("skipb", skipb), zap.Any("skipc", skipc), zap.Any("%change", change), zap.Any("result", result))

	if result.TradeType == TradeTypeShort {
		logger.Warn(ctx, "skipping shorts", zap.Any("result", result))

		return
	}

	if !(skipa || skipc) {
		s.placeTrade(ctx, result)
	}
}

func (s *system) placeTrade(ctx context.Context, result *TradeParams) {
	if result != nil {
		// TODO(oblessing): don't allow close at the same price, throw error so moderator can close it.
		if result.TakeProfitAt == result.OpenTradeAt || result.StopLossAt == result.OpenTradeAt {
			m := "same open + stop"
			if result.TakeProfitAt == result.OpenTradeAt {
				m = "same open + profit"
			}
			logger.Error(ctx, "trade mismatch", zap.String("mismatch", m), zap.Any("t", result))

			return
		}

		// acquire lock
		s.rw.Lock()
		defer s.rw.Unlock()

		// Check if we have open trade.
		// TODO: support opening of multiple positions.
		if _, ok := read(result.Pair); ok {
			// logger.Warn(ctx, "already have an open trade", zap.Any("ignored", result))

			return
		}

		// open trade, retry 10 times before closing. (we must try to place trade)
		for count := 1; count <= 10; count += 1 {
			trd, err := s.orderService.PlaceTrade(ctx, *result)
			if err != nil {
				logger.Warn(ctx, "failed place order, retrying", zap.Any("ignored", result), zap.Int("count", count), zap.Error(err))
				continue
			}

			result.OrderID = trd.OrderID

			write(result.Pair, result)

			break
		}
	}
}

func findNumberOfDecimal(v string) uint8 {
	data := strings.Split(v, ".")
	if len(data) == 2 {
		var result uint8 = 0
		for _, i := range data[1] {
			if i == '0' {
				result += 1
			} else {
				return result + 1
			}
		}
	}

	return 0
}

// RoundToDecimalPoint take an amount then rounds it to the upper 2 decimal point if the value is more than 2 decimal point.
func RoundToDecimalPoint(amount float64, precision uint8) float64 {
	str := "%." + fmt.Sprintf("%v", precision) + "f"
	amountString := fmt.Sprintf(str, amount)

	res, _ := strconv.ParseFloat(amountString, 64)
	return res
}

func (s *system) tradeClosed(pair Pair) {
	remove(pair)
}

func (s *system) tryClosing(ctx context.Context, candle *Candle) {
	params, ok := read(candle.Pair)
	if !ok {
		// we currently don't have an active trade
		return
	}

	var err error
	var closedTrade bool

	// try closing based on trade type.
	switch params.TradeType {
	case TradeTypeLong:
		if candle.Close >= params.TakeProfitAtV() {
			if params.AutomaticClose {
				closedTrade = true
				break
			}

			closedTrade, err = s.orderService.CloseTrade(ctx, SellParams{
				IsStopLoss:  false,
				SellTradeAt: candle.Close,
				PL:          candle.Close - params.OpenTradeAtV(),
				Pair:        candle.Pair,
				TradeSize:   params.TradeSize,
				OrderID:     params.OrderID,
				TradeType:   params.TradeType,
			})
		} else if candle.Close <= params.StopLossAtV() {
			if params.AutomaticClose {
				closedTrade = true
				break
			}

			closedTrade, err = s.orderService.CloseTrade(ctx, SellParams{
				IsStopLoss:  true,
				SellTradeAt: candle.Close,
				PL:          candle.Close - params.OpenTradeAtV(),
				Pair:        candle.Pair,
				TradeSize:   params.TradeSize,
				OrderID:     params.OrderID,
				TradeType:   params.TradeType,
			})
		}
	case TradeTypeShort:
		if candle.Close <= params.TakeProfitAtV() {
			if params.AutomaticClose {
				closedTrade = true
				break
			}

			closedTrade, err = s.orderService.CloseTrade(ctx, SellParams{
				IsStopLoss:  false,
				SellTradeAt: candle.Close,
				PL:          params.OpenTradeAtV() - candle.Close,
				Pair:        candle.Pair,
				TradeSize:   params.TradeSize,
				OrderID:     params.OrderID,
				TradeType:   params.TradeType,
			})
		} else if candle.Close >= params.StopLossAtV() {
			if params.AutomaticClose {
				closedTrade = true
				break
			}

			closedTrade, err = s.orderService.CloseTrade(ctx, SellParams{
				IsStopLoss:  true,
				SellTradeAt: candle.Close,
				PL:          params.OpenTradeAtV() - candle.Close,
				Pair:        candle.Pair,
				TradeSize:   params.TradeSize,
				OrderID:     params.OrderID,
				TradeType:   params.TradeType,
			})
		}
	}

	if err != nil {
		logger.Error(ctx, "ea_trader: error occurred while attempting to close trade", zap.Error(err), zap.Any("p", params))
		return
	}

	if closedTrade {
		s.tradeClosed(candle.Pair)
	}
}

func convertToHeikinAshi(older *Candle, newer *Candle) *Candle {
	if newer == nil {
		return nil
	}
	if older == nil {
		older = newer
	}

	// close
	newClose := 0.25 * (newer.Open + newer.Close + newer.High + newer.Low)
	newOpen := 0.5 * (older.Open + older.Close)
	newLow := lowest(newer.Low, newOpen, newClose)
	newHigh := highest(newer.High, newOpen, newClose)

	result := &Candle{
		Pair:      newer.Pair,
		High:      newHigh,
		Low:       newLow,
		Open:      newOpen,
		Close:     newClose,
		Volume:    newer.Volume,
		OtherData: map[string]float64{},
		Time:      newer.Time,
		Closed:    newer.Closed,
	}

	return result
}

func lowest(arr ...float64) float64 {
	value := arr[0]
	for _, v := range arr {
		if v < value {
			value = v
		}
	}
	return value
}

func highest(arr ...float64) float64 {
	value := arr[0]
	for _, v := range arr {
		if v > value {
			value = v
		}
	}
	return value
}

func read(key Pair) (*TradeParams, bool) {
	result, ok := activeTrades.Load(key)
	if !ok {
		return nil, false
	}

	return result.(*TradeParams), ok
}

func remove(key Pair) {
	activeTrades.Delete(key)
}

func write(key Pair, data *TradeParams) {
	activeTrades.Store(key, data)
}
