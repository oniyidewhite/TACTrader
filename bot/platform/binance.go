package platform

import (
	"github.com/adshao/go-binance/v2"
	"github.com/oblessing/artisgo/bot"
	"github.com/oblessing/artisgo/expert"
	log2 "log"
	"os"
	"strconv"
)

// consts
const (
	logPrefix = "binance:\t"
)

// vars
var log *log2.Logger

// structs
type Binance struct {
	binance.Client
	transform expert.Transform

	expert expert.Trader
}

// init
func init() {
	log = log2.New(os.Stdout, logPrefix, log2.LstdFlags|log2.Lshortfile)
}

// methods
func (r *Binance) OnCreate(config *bot.Config) {
	binance.UseTestnet = config.IsTest
	wsKlineHandler := func(event *binance.WsKlineEvent) {
		// pass result to expert Trader
		r.expert.Record(convert(event), r.transform)
	}

	errHandler := func(err error) {
		log.Println(err)
	}

	// try to connect to binance
	doneC, _, err := binance.WsKlineServe(config.Pair, config.Period, wsKlineHandler, errHandler)
	if err != nil {
		log.Println(err)
		return
	}

	go func() {
		<-doneC
	}()
}

// check if we can close this trade.
// if trade doesn't exist we still return false
func convert(kline *binance.WsKlineEvent) *expert.Candle {
	high, err := parse(kline.Kline.High)
	if err != nil {
		return nil
	}
	low, err := parse(kline.Kline.Low)
	if err != nil {
		return nil
	}
	open, err := parse(kline.Kline.Open)
	if err != nil {
		return nil
	}
	cl, err := parse(kline.Kline.Close)
	if err != nil {
		return nil
	}
	vol, err := parse(kline.Kline.Volume)
	if err != nil {
		return nil
	}

	return &expert.Candle{
		Pair:   expert.Pair(kline.Symbol),
		High:   high,
		Low:    low,
		Open:   open,
		Close:  cl,
		Volume: vol,
		Time:   kline.Time,
		Closed: kline.Kline.IsFinal,
	}
}

func parse(value string) (float64, error) {
	return strconv.ParseFloat(value, 64)
}

// functions
func New() {

}
