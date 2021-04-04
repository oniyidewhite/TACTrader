package main

import (
	"github.com/adshao/go-binance/v2"
	bot2 "github.com/oblessing/artisgo/bot"
	"github.com/oblessing/artisgo/bot/platform"
	"github.com/oblessing/artisgo/bot/store/mongo"
	"github.com/oblessing/artisgo/expert"
	log2 "log"
	"os"
)

// this params would be injected
var (
	BinanceApiKey    string = ""
	BinanceSecretKey string = ""
	DatabaseUri      string = "mongodb://admin:password@127.0.0.1:27017/admin"
	logger           *log2.Logger
	logPrefix        = "app:\t"
)

const (
	MIN float64 = -1 << 31
	MAX float64 = 1 << 31
)

func init() {
	logger = log2.New(os.Stdout, logPrefix, log2.LstdFlags|log2.Lshortfile)
}

func main() {
	// create all the supported symbols
	storage, err := mongo.NewMongoInstance(DatabaseUri)
	if err != nil {
		panic(err)
	}
	eaConfig := &expert.Config{
		Size:       18,
		BuyAction:  buy,
		SellAction: sell,
		Storage:    expert.NewDataSource(storage),
	}
	ea, err := expert.NewTrader(eaConfig)
	if err != nil {
		panic(err)
	}
	pConfig := &platform.Config{
		Client:    binance.NewClient(BinanceApiKey, BinanceSecretKey),
		Transform: transform,
		Expert:    ea,
	}

	bConfigHot := &bot2.Config{
		Pair:   "HOTUSDT",
		Period: "5m",
		IsTest: false,
	}

	bConfigEth := &bot2.Config{
		Pair:   "ETHBUSD",
		Period: "5m",
		IsTest: false,
	}

	bConfigAlc := &bot2.Config{
		Pair:   "ALICEUSDT",
		Period: "5m",
		IsTest: false,
	}

	bot := platform.New(pConfig)

	go bot.OnCreate(bConfigHot)
	go bot.OnCreate(bConfigEth)
	go bot.OnCreate(bConfigAlc)

	bot.OnCreate(bConfigEth)
}

func transform(candles []*expert.Candle) *expert.TradeParams {
	high1, high2 := MIN, MIN
	low1, low2 := MAX, MAX

	current := candles[0].Close

	for _, c := range candles {
		if c.Low < low1 {
			low2 = low1
			low1 = c.Low
		}

		if c.High > high1 {
			high2 = high1
			high1 = c.High
		}
	}

	avgHigh := (high1 + high2) / 2
	avgLow := (low1 + low2) / 2

	rate := ((current - avgLow) / (avgHigh - avgLow)) * 100

	result := &expert.TradeParams{
		OpenTradeAt:  current,
		TakeProfitAt: avgHigh,
		StopLossAt:   avgLow,
		Rating:       int(rate),
		Pair:         candles[0].Pair,
	}
	return result
}

func buy(params *expert.TradeParams) bool {
	// TODO: Connect to binance API
	logger.Println("Initiated Buy (%s) At: %f", params.Pair, params.OpenTradeAt)
	return true
}
func sell(params *expert.SellParams) bool {
	// TODO: Connect to binance API
	if params.IsStopLoss {
		logger.Println("Initiated Sell-Loss (%s) At: %f", params.Pair, params.PL)
	} else {
		logger.Println("Initiated Take-Profit (%s) At: %f", params.Pair, params.PL)
	}
	return true
}
