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
	MIN float64 = -1 << 63
	MAX float64 = 1 << 63
)

func init() {
	logger = log2.New(os.Stdout, logPrefix, log2.LstdFlags|log2.Lshortfile)
}

func main() {
	// create all the supported symbols
	bot := initTradeBot()

	//botOtherTradeList(bot)

	// Main
	bConfigBnb := &bot2.Config{
		Pair:   "BNBEUR",
		Period: "1m",
		IsTest: false,
	}

	bot.OnCreate(bConfigBnb)
}

// myCustomTransform uses an array of candles find the lowest and highest that calculate it's percentage to the current
// returns that result as it's rating
// Use the MA to determine it it's good buy or not
func myCustomTransform(candles []*expert.Candle) *expert.TradeParams {
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

	_ = (high1 + high2) / 2
	_ = (low1 + low2) / 2

	rate := ((current - low1) / (high1 - low1)) * 100

	result := &expert.TradeParams{
		OpenTradeAt:  current,
		TakeProfitAt: high1,
		StopLossAt:   low1,
		Rating:       int(rate),
		Pair:         candles[0].Pair,
	}
	return result
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

func initTradeBot() bot2.TradeBot {
	storage, err := mongo.NewMongoInstance(DatabaseUri)
	if err != nil {
		logger.Fatal(err)
	}
	eaConfig := &expert.Config{
		Size:            18,
		BuyAction:       buy,
		SellAction:      sell,
		Storage:         expert.NewDataSource(storage),
		CalculateAction: calculateActions(),
	}
	ea, err := expert.NewTrader(eaConfig)
	if err != nil {
		panic(err)
	}
	pConfig := &platform.Config{
		Client:    binance.NewClient(BinanceApiKey, BinanceSecretKey),
		Transform: myCustomTransform,
		Expert:    ea,
	}
	bot := platform.New(pConfig)
	return bot
}

func calculateActions() []*expert.CalculateAction {
	return []*expert.CalculateAction{
		{
			Name: "MA36",
			Size: 36,
			Action: func(candles []*expert.Candle) float64 {
				var sum float64 = 0
				for _, i := range candles {
					sum += i.Close
				}

				return sum / float64(len(candles))
			},
		},
		{
			Name: "RSI6",
			Size: 6,
			Action: func(candles []*expert.Candle) float64 {
				var sumUp float64 = 0
				var sumDown float64 = 0
				for _, i := range candles {
					if i.IsUp() {
						sumUp += i.Close - i.Open
					} else {
						sumDown += i.Open - i.Close
					}
				}

				// TODO(oblessing): Complete RSI

				return 0
			},
		},
	}
}

func botOtherTradeList(bot bot2.TradeBot) {
	bConfigHot := &bot2.Config{
		Pair:   "HOTUSDT",
		Period: "1m",
		IsTest: false,
	}

	bConfigEth := &bot2.Config{
		Pair:   "ETHBUSD",
		Period: "1m",
		IsTest: false,
	}

	bConfigAlc := &bot2.Config{
		Pair:   "ALICEBUSD",
		Period: "1m",
		IsTest: false,
	}

	bConfigDoge := &bot2.Config{
		Pair:   "DOGEBUSD",
		Period: "1m",
		IsTest: false,
	}

	go bot.OnCreate(bConfigHot)
	go bot.OnCreate(bConfigDoge)
	go bot.OnCreate(bConfigEth)
	go bot.OnCreate(bConfigAlc)
}
