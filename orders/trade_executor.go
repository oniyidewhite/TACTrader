package orders

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
	"go.uber.org/zap"

	settings "github.com/oblessing/artisgo"
	"github.com/oblessing/artisgo/expert"
	"github.com/oblessing/artisgo/logger"
)

type binanceAdapter struct {
	client     *futures.Client
	isTestMode bool
}

type OrderService interface {
	PlaceTrade(ctx context.Context, params expert.TradeParams) (expert.TradeData, error)
	CloseTrade(ctx context.Context, params expert.SellParams) (bool, error)
}

func NewAdapter(config settings.Config) *binanceAdapter {
	binance.UseTestnet = config.IsTestMode()
	return &binanceAdapter{
		client:     binance.NewFuturesClient(config.BinanceApiKey, config.BinanceSecretKey),
		isTestMode: config.IsTestMode(),
	}
}

func (b *binanceAdapter) PlaceTrade(ctx context.Context, params expert.TradeParams) (expert.TradeData, error) {
	ctx = logger.With(ctx,
		zap.Any("tradeType", params.TradeType),
		zap.Float64("openTradeAt", params.OpenTradeAt),
		zap.Float64("takeProfitAt", params.TakeProfitAt),
		zap.Float64("stopLossAt", params.StopLossAt),
		zap.Any("tradeSize", params.TradeSize),
		zap.Int("rating", params.Rating))

	if b.isTestMode {
		return expert.TradeData{}, nil
	}

	switch params.TradeType {
	case expert.TradeTypeLong:
		return b.placeLong(ctx, params)
	case expert.TradeTypeShort:
		return b.placeShort(ctx, params)
	default:
		return expert.TradeData{}, errors.New("unsupported trade tyep")
	}
}

func (b *binanceAdapter) CloseTrade(ctx context.Context, params expert.SellParams) (bool, error) {
	ctx = logger.With(ctx,
		zap.Any("pair", params.Pair),
		zap.Any("tradeType", params.TradeType),
		zap.Bool("isStopLoss", params.IsStopLoss),
		zap.Float64("sellTradeAt", params.SellTradeAt),
		zap.String("orderID", params.OrderID),
		zap.Float64("pl", params.PL),
	)

	if b.isTestMode {
		if params.IsStopLoss {
			logger.Info(ctx, "stop loss")
			return true, nil
		}

		logger.Info(ctx, "take profit")
		return true, nil
	}

	orderID, err := strconv.ParseInt(params.OrderID, 10, 64)
	if err != nil {
		return false, err
	}

	_, err = b.client.NewCancelOrderService().Symbol(string(params.Pair)).OrderID(orderID).Do(ctx)
	if err != nil {
		return false, err
	}

	return true, nil
}

// EnableIsolatedTrading tells binance that this pair should be traded in isolated mode.
func (b *binanceAdapter) EnableIsolatedTrading(ctx context.Context, pair expert.Pair) error {
	err := b.client.NewChangeMarginTypeService().MarginType(futures.MarginTypeIsolated).Symbol(string(pair)).Do(ctx)
	return err
}

// SetLeverage tells binance to use a specific amount for this trade.
func (b *binanceAdapter) SetLeverage(ctx context.Context, pair expert.Pair) error {
	_, err := b.client.NewChangeLeverageService().Symbol(string(pair)).Leverage(10).Do(ctx)
	return err
}

func (b *binanceAdapter) placeLong(ctx context.Context, params expert.TradeParams) (expert.TradeData, error) {
	res, err := b.client.NewCreateOrderService().
		Symbol(string(params.Pair)).
		PositionSide(futures.PositionSideTypeLong).
		Price(fmt.Sprintf("%f", params.OpenTradeAt)).
		StopPrice(fmt.Sprintf("%f", params.StopLossAt)).
		Quantity(params.TradeSize).
		Side(futures.SideTypeBuy).
		Type(futures.OrderTypeStop). //OrderTypeStopMarket, ClosePosition(true).
		TimeInForce(futures.TimeInForceTypeFOK).
		NewOrderResponseType(futures.NewOrderRespTypeRESULT).
		Do(ctx)
	if err != nil {
		return expert.TradeData{}, err
	}

	logger.Info(ctx, "order: placed long", zap.Any("response", res))

	return expert.TradeData{
		OrderID:       fmt.Sprintf("%d", res.OrderID),
		ClientOrderID: res.ClientOrderID,
	}, err
}

func (b *binanceAdapter) placeShort(ctx context.Context, params expert.TradeParams) (expert.TradeData, error) {
	res, err := b.client.NewCreateOrderService().
		Symbol(string(params.Pair)).
		PositionSide(futures.PositionSideTypeShort).
		Price(fmt.Sprintf("%f", params.OpenTradeAt)).
		StopPrice(fmt.Sprintf("%f", params.StopLossAt)).
		Quantity(params.TradeSize).
		Side(futures.SideTypeSell).
		Type(futures.OrderTypeStop). //OrderTypeStopMarket, ClosePosition(true).
		TimeInForce(futures.TimeInForceTypeFOK).
		NewOrderResponseType(futures.NewOrderRespTypeRESULT).
		Do(ctx)
	if err != nil {
		return expert.TradeData{}, err
	}

	logger.Info(ctx, "order: placed short", zap.Any("response", res))

	return expert.TradeData{
		OrderID:       fmt.Sprintf("%d", res.OrderID),
		ClientOrderID: res.ClientOrderID,
	}, err
}
