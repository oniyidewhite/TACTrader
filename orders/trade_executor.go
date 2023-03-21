package orders

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	settings "github.com/oblessing/artisgo"
	"github.com/oblessing/artisgo/expert"
	"github.com/oblessing/artisgo/logger"
	"github.com/oblessing/artisgo/strategy"
)

type binanceAdapter struct {
	client     *futures.Client
	isTestMode bool
}

type OrderService interface {
	PlaceTrade(ctx context.Context, params expert.TradeParams) (expert.TradeData, error)
	CloseTrade(ctx context.Context, params expert.SellParams) (bool, error)
	UpdateConfiguration(ctx context.Context, pairs ...expert.Pair) error
}

func NewAdapter(config settings.Config) *binanceAdapter {
	//binance.UseTestnet = config.IsTestMode()
	return &binanceAdapter{
		client:     binance.NewFuturesClient(config.BinanceApiKey, config.BinanceSecretKey),
		isTestMode: config.IsTestMode(),
	}
}

// UpdateConfiguration runs any needed configuration to the trade executor, returns only valid paris that we were able to configure.
func (b *binanceAdapter) UpdateConfiguration(ctx context.Context, pairs ...strategy.PairConfig) ([]strategy.PairConfig, error) {
	// There's no need to update config in test mode.
	if b.isTestMode {
		return pairs, nil
	}

	var updatedPairs []strategy.PairConfig
	validPairs := make(chan strategy.PairConfig)
	g := errgroup.Group{}
	for _, pair := range pairs {
		p := pair
		g.Go(func() error {
			err := b.enableIsolatedTrading(ctx, expert.Pair(p.Pair))
			if err != nil {
				// Do not log unsupported pairs [just silently ignore]
				logger.Error(ctx, "skipping: enableIsolatedTrading", zap.Any("symbol", p.Pair), zap.Error(err))
				//return err
			}
			err = b.setLeverage(ctx, expert.Pair(p.Pair))
			if err != nil {
				// Do not log unsupported pairs [just silently ignore]
				logger.Error(ctx, "skipping: unable to update leverage", zap.Any("symbol", p.Pair), zap.Error(err))
			} else {
				validPairs <- p
			}
			return err
		})
	}

	g.Go(func() error {
		<-time.After(10 * time.Second)
		close(validPairs)
		return nil
	})

	for v := range validPairs {
		updatedPairs = append(updatedPairs, v)
	}

	return updatedPairs, nil
}

func (b *binanceAdapter) PlaceTrade(ctx context.Context, params expert.TradeParams) (expert.TradeData, error) {
	ctx = logger.With(ctx,
		zap.Any("p", params.Pair),
		zap.Any("ty", params.TradeType),
		zap.Any("ot", params.OpenTradeAt),
		zap.Any("tp", params.TakeProfitAt),
		zap.Any("sl", params.StopLossAt),
		zap.Any("tz", params.TradeSize),
		zap.Any("atr", params.Attribs))

	if params.OpenTradeAt == params.TakeProfitAt {
		return expert.TradeData{}, errors.New("can not open a trade at the take profit position")
	}

	if params.OpenTradeAt == params.StopLossAt {
		return expert.TradeData{}, errors.New("can not open a trade at the take stoploss position")
	}

	if b.isTestMode {
		logger.Info(ctx, "placed order")
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
		zap.Any("p", params.Pair),
		zap.Any("ty", params.TradeType),
		zap.Bool("isl", params.IsStopLoss),
		zap.Float64("sa", params.SellTradeAt),
		zap.String("oid", params.OrderID),
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

	var side = futures.SideTypeSell
	if params.TradeType == expert.TradeTypeShort {
		side = futures.SideTypeBuy
	}

	res, err := b.client.NewCreateOrderService().Symbol(string(params.Pair)).Side(side).Quantity(params.TradeSize).Type(futures.OrderTypeMarket).Do(ctx) //.NewCancelOrderService().Symbol(string(params.Pair)).OrderID(orderID).Do(ctx)
	if err != nil {
		if params.IsStopLoss {
			// The trade already close, trust me. lol 🌚
			return true, nil
		}

		return false, err
	}

	logger.Info(ctx, "order: close trade", zap.Any("response", res))

	return true, nil
}

// EnableIsolatedTrading tells binance that this pair should be traded in isolated mode.
func (b *binanceAdapter) enableIsolatedTrading(ctx context.Context, pair expert.Pair) error {
	err := b.client.NewChangeMarginTypeService().MarginType(futures.MarginTypeIsolated).Symbol(string(pair)).Do(ctx)
	return err
}

// SetLeverage tells binance to use a specific amount for this trade.
func (b *binanceAdapter) setLeverage(ctx context.Context, pair expert.Pair) error {
	_, err := b.client.NewChangeLeverageService().Symbol(string(pair)).Leverage(20).Do(ctx)
	return err
}

func (b *binanceAdapter) placeLong(ctx context.Context, params expert.TradeParams) (expert.TradeData, error) {
	res, err := b.client.NewCreateOrderService().
		Symbol(string(params.Pair)).
		PositionSide(futures.PositionSideTypeBoth).
		Side(futures.SideTypeBuy).
		Price(fmt.Sprintf("%s", params.OpenTradeAt)).
		//StopPrice(fmt.Sprintf("%s", params.TakeProfitAt)).
		Quantity(params.TradeSize).
		Side(futures.SideTypeBuy).
		Type(futures.OrderTypeLimit).
		//Type(futures.OrderTypeTakeProfit). //OrderTypeStopMarket, ClosePosition(true).
		TimeInForce(futures.TimeInForceTypeFOK).
		NewOrderResponseType(futures.NewOrderRespTypeRESULT).
		Do(ctx)
	if err != nil {
		return expert.TradeData{}, err
	}

	if res.Status == "EXPIRED" {
		return expert.TradeData{}, errors.New("trade expired")
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
		PositionSide(futures.PositionSideTypeBoth).
		Side(futures.SideTypeSell).
		Price(fmt.Sprintf("%s", params.OpenTradeAt)).
		//StopPrice(fmt.Sprintf("%s", params.TakeProfitAt)).
		Quantity(params.TradeSize).
		Side(futures.SideTypeSell).
		Type(futures.OrderTypeLimit).
		//Type(futures.OrderTypeTakeProfit). //OrderTypeStopMarket, ClosePosition(true).
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
