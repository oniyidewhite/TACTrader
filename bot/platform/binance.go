package platform

import (
	"context"
	"errors"
	"github.com/adshao/go-binance/v2"
	"github.com/oblessing/artisgo/bot"
	"github.com/oblessing/artisgo/expert"
	"github.com/oblessing/artisgo/logger"
	"go.uber.org/zap"
	log2 "log"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	logPrefix = "binance:\t"
)

var log *log2.Logger

// myBinance represent My Binance API configuration
type myBinance struct {
	// Crypto pairs we are watching
	pairs []bot.PairConfig
	// Know if the trading bot has started trading
	hasStarted bool
	// Helps sync start
	state sync.Once

	expert expert.Trader
}

func (r *myBinance) WatchAndTrade(config ...bot.PairConfig) error {
	if r.hasStarted {
		return errors.New("bot has already started")
	}

	r.pairs = append(r.pairs, config...)

	return nil
}

func (r *myBinance) StartTrading() error {
	if r.hasStarted {
		return errors.New("bot has already started")
	}

	r.state.Do(func() {
		r.hasStarted = true

		errHandler := func(err error) {
			log.Println(err)
		}

		ctx := context.Background()

		// Start all the current pairs
		for _, p := range r.pairs {
			p := p
			go func() {
				wsKlineHandler := func(event *binance.WsKlineEvent) {
					// pass result to expert Trader
					r.expert.Record(convert(event), p.Strategy, expert.RecordConfig{
						LotSize:        p.LotSize,
						RatioToOne:     p.RatioToOne,
						OverrideParams: p.OverrideParams,
						TradeSize:      p.TradeSize,
						Spread:         p.Spread,
					})
				}

				// We restart if we encounter an error.
				for {
					logger.Info(ctx, "starting watcher", zap.String("pair", p.Pair), zap.String("period", p.Period))
					doneC, _, err := binance.WsKlineServe(p.Pair, p.Period, wsKlineHandler, errHandler)
					if err != nil {
						logger.Error(ctx, "an error occurred", zap.Error(err))
						<-time.After(3 * time.Second)
						continue
					}

					<-doneC
				}
			}()
		}
	})

	// Lock
	<-make(chan struct{})
	return nil
}

type Config struct {
	Expert expert.Trader
}

// init
func init() {
	log = log2.New(os.Stdout, logPrefix, log2.LstdFlags|log2.Lshortfile)
}

// check if we can close this trade.
// if trade doesn't exist we still return false
func convert(kline *binance.WsKlineEvent) *expert.Candle {
	high, err := parseString(kline.Kline.High)
	if err != nil {
		return nil
	}
	low, err := parseString(kline.Kline.Low)
	if err != nil {
		return nil
	}
	open, err := parseString(kline.Kline.Open)
	if err != nil {
		return nil
	}
	cl, err := parseString(kline.Kline.Close)
	if err != nil {
		return nil
	}
	vol, err := parseString(kline.Kline.Volume)
	if err != nil {
		return nil
	}

	return &expert.Candle{
		Pair:      expert.Pair(kline.Symbol),
		High:      high,
		Low:       low,
		Open:      open,
		Close:     cl,
		Volume:    vol,
		Time:      kline.Time,
		Closed:    kline.Kline.IsFinal,
		OtherData: map[string]float64{},
	}
}
func parseString(value string) (float64, error) {
	return strconv.ParseFloat(value, 64)
}

// NewBinanceTrader return a new instance of binance trader.
func NewBinanceTrader(config Config) bot.Trader {
	return &myBinance{
		expert: config.Expert,
	}
}
