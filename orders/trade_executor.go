package orders

import (
	"context"
	"fmt"
	"github.com/adshao/go-binance/v2"
	trade "github.com/oblessing/artisgo/bot"
	"github.com/oblessing/artisgo/expert"
	"github.com/oblessing/artisgo/logger"
	"go.uber.org/zap"
)

var binanceApiKey = ""
var binanceSecretKey = ""

var client *binance.Client

var TradeList = []trade.PairConfig{
	{
		Pair:           "BNBEUR",
		Period:         "1m",
		Strategy:       trade.MyOldCustomTransform,
		OverrideParams: true,
		LotSize:        1.3,
		RatioToOne:     3,
		TradeSize:      "0.114",
	},
	{
		Pair:           "ETHEUR",
		Period:         "1m",
		Strategy:       trade.MyOldCustomTransform,
		OverrideParams: true,
		LotSize:        16,
		RatioToOne:     3,
		TradeSize:      "0.0166",
	},
	{
		Pair:           "TRXEUR",
		Period:         "1m",
		Strategy:       trade.MyOldCustomTransform,
		OverrideParams: true,
		LotSize:        0.00022,
		RatioToOne:     3,
		TradeSize:      "707",
	},
	{
		Pair:           "DOGEEUR",
		Period:         "1m",
		Strategy:       trade.MyOldCustomTransform,
		OverrideParams: true,
		LotSize:        0.0005,
		RatioToOne:     3,
		TradeSize:      "318",
	},
	{
		Pair:           "XRPUSDT",
		Period:         "1h",
		Strategy:       trade.MyOldCustomTransform,
		OverrideParams: true,
		LotSize:        0.0020,
		RatioToOne:     3,
		TradeSize:      "58",
	},
}

func init() {
	client = binance.NewClient(binanceApiKey, binanceSecretKey)
}

func Buy(params *expert.TradeParams) bool {
	ctx := context.Background()

	ctx = logger.With(ctx,
		zap.Any("Pair", params.Pair),
		zap.Any("TradeSize", params.TradeSize),
		zap.Float64("OpenTradeAt", params.OpenTradeAt),
		zap.Float64("TakeProfitAt", params.TakeProfitAt),
		zap.Float64("StopLossAt", params.StopLossAt),
		zap.Int("Rating", params.Rating))

	switch params.Pair {
	case "":
		// TODO: ADD our checks
		res, err := client.NewCreateOrderService().
			Symbol(string(params.Pair)).
			Side(binance.SideTypeBuy).
			Price(fmt.Sprintf("%v", params.OpenTradeAt)).
			Quantity(params.TradeSize).
			TimeInForce(binance.TimeInForceTypeGTC).
			Type(binance.OrderTypeLimit).
			Do(context.Background())
		if err != nil {
			logger.Error(ctx, "error placing order", zap.Error(err))
			return false
		}

		params.OrderID = res.OrderID
		ctx = logger.With(ctx, zap.Int64("order_id", params.OrderID))
	default:
		logger.Info(ctx, "buyAction not supported: marked a buy order")
		return true
	}

	logger.Info(ctx, "successfully placed buy order")
	return true
}
func Sell(params *expert.SellParams) bool {
	ctx := context.Background()

	ctx = logger.With(ctx,
		zap.Any("Pair", params.Pair),
		zap.Bool("IsStopLoss", params.IsStopLoss),
		zap.Float64("SellTradeAt", params.SellTradeAt),
		zap.Int64("OrderID", params.OrderID),
		zap.Float64("PL", params.PL),
	)
	switch params.Pair {
	case "":
		// TODO: ADD our checks
		res, err := client.NewCreateOrderService().
			Symbol(string(params.Pair)).
			Side(binance.SideTypeSell).
			Price(fmt.Sprintf("%v", params.SellTradeAt)).
			Quantity(params.TradeSize).
			TimeInForce(binance.TimeInForceTypeGTC).
			Type(binance.OrderTypeLimit).
			Do(context.Background())
		if err != nil {
			logger.Error(ctx, "error placing order", zap.Error(err))
			return false
		}
		ctx = logger.With(ctx, zap.Int64("sell_order_id", res.OrderID))

	default:
		if params.IsStopLoss {
			logger.Info(ctx, "stop loss")
			return true
		}
		logger.Info(ctx, "take profit")
		return true
	}

	logger.Info(ctx, "successfully placed sell order")
	return true
}
