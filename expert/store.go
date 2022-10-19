package expert

import (
	"context"
	"time"

	"github.com/oblessing/artisgo/bot/store"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type mapper struct{}

type storage struct {
	store  store.Database
	mapper *mapper
}

func (m *storage) FetchCandles(ctx context.Context, pair Pair, size int) ([]*Candle, error) {
	response, err := m.store.Fetch(ctx, string(pair), size)
	if err != nil {
		return []*Candle{}, err
	}

	result := []*Candle{}
	for _, v := range response {
		result = append(result, m.mapper.convertFrom(v))
	}

	return result, nil
}

func (m *storage) Persist(ctx context.Context, candle *Candle) error {
	data := m.mapper.convertTo(candle)
	return m.store.Save(ctx, data)
}

func (m *mapper) convertFrom(candle *store.BotData) *Candle {
	return &Candle{
		Pair:      Pair(candle.Pair),
		High:      candle.Candle.High,
		Low:       candle.Candle.Low,
		Open:      candle.Candle.Open,
		Close:     candle.Candle.Close,
		Volume:    candle.Candle.Vol,
		Time:      candle.Date.Unix(),
		Closed:    candle.IsClosed,
		OtherData: candle.Others,
	}
}

func (m *mapper) convertTo(candle *Candle) *store.BotData {
	return &store.BotData{
		Candle: &store.Candle{
			Open:  candle.Open,
			Close: candle.Close,
			High:  candle.High,
			Low:   candle.Low,
			Vol:   candle.Volume,
		},
		Others:   candle.OtherData,
		IsClosed: candle.Closed,
		Date:     time.Unix(candle.Time/1000, 0),
		Pair:     string(candle.Pair),
		Id:       primitive.NewObjectID(),
	}
}

func NewDataSource(database store.Database) DataSource {
	return &storage{
		store:  database,
		mapper: &mapper{}}
}
