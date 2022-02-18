package main

import (
	"github.com/adshao/go-binance/v2"
	trade "github.com/oblessing/artisgo/bot"
	"github.com/oblessing/artisgo/bot/platform"
	"github.com/oblessing/artisgo/bot/store"
	"github.com/oblessing/artisgo/bot/store/mongo"
	"github.com/oblessing/artisgo/expert"
	log2 "log"
	"os"
)

// this params would be injected
var (
	binanceApiKey    = "rLNxccORz0Erumx6JfA7RAHyykQUklSQ338gANKQdizcRBT1BpxPwu2QD5nuO8Jr"
	binanceSecretKey = ""
	DatabaseUri      = "mongodb://user:password@localhost:27017"
	logger           *log2.Logger
	logPrefix        = "app:\t"
)

func init() {
	logger = log2.New(os.Stdout, logPrefix, log2.LstdFlags|log2.Lshortfile)
}

func main() {
	// create database
	storage, err := mongo.NewMongoInstance(DatabaseUri)
	if err != nil {
		logger.Fatal(err)
	}

	// Build Expert trader
	trader := buildBinanceTrader(false, storage, buy, sell, trade.GetDefaultAnalysis(), binanceApiKey, binanceSecretKey)

	// create all the supported symbols
	if err = trader.WatchAndTrade(
		trade.PairConfig{
			Pair:     "BNBEUR",
			Period:   "1m",
			Strategy: trade.MyOldCustomTransform,
		},
		trade.PairConfig{
			Pair:     "ETHEUR",
			Period:   "1m",
			Strategy: trade.MyOldCustomTransform,
		},
		trade.PairConfig{
			Pair:     "TRXEUR",
			Period:   "1m",
			Strategy: trade.MyOldCustomTransform,
		},
		trade.PairConfig{
			Pair:      "DOGEEUR",
			Period:    "1m",
			Strategy:  trade.MyOldCustomTransform,
			TradeSize: "2",
		}); err != nil {
		logger.Fatal(err)
	}

	//watchlist = append(watchlist, &trade.PairConfig{
	//	Pair:   "XRPUSDT",
	//	Period: "1h",
	//	IsTest: false,
	//})
	//

	if err = trader.StartTrading(); err != nil {
		logger.Fatal(err)
	}
}

func buy(params *expert.TradeParams) bool {
	// TODO: Connect to binance API
	logger.Printf("(%s): Buy-At:%f\n", params.Pair, params.OpenTradeAt)
	return true
}
func sell(params *expert.SellParams) bool {
	// TODO: Connect to binance API
	if params.IsStopLoss {
		logger.Printf("(%s): Stop-Loss:%f\n", params.Pair, params.PL)
	} else {
		logger.Printf("(%s): Take-Profit:%f\n", params.Pair, params.PL)
	}
	return true
}

func buildBinanceTrader(testMode bool, storage store.Database, buyAction expert.BuyAction, sellAction expert.SellAction, defaultAnalysis []*expert.CalculateAction, binanceApiKey string, binanceSecretKey string) trade.Trader {
	binance.UseTestnet = testMode

	return platform.NewBinanceTrader(platform.Config{
		Client: binance.NewClient(binanceApiKey, binanceSecretKey),
		Expert: expert.NewTrader(&expert.Config{
			Size:            18, //TODO: should be tied to strategy
			BuyAction:       buyAction,
			SellAction:      sellAction,
			Storage:         expert.NewDataSource(storage),
			DefaultAnalysis: defaultAnalysis,
		}),
	})
}
