package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BotData struct {
	Others   map[string]float64 `bson:"others"`
	Candle   *Candle            `bson:"candle"`
	IsClosed bool               `bson:"is_closed"`
	Date     time.Time          `bson:"date"`
	Pair     string             `bson:"pair"`
	Id       primitive.ObjectID `bson:"_id"`
}

type Candle struct {
	Open  float64 `bson:"open"`
	Close float64 `bson:"close"`
	High  float64 `bson:"high"`
	Low   float64 `bson:"low"`
	Vol   float64 `bson:"vol"`
}

type Database interface {
	// Save date to database
	Save(context.Context, *BotData) error
	// Fetch retrieves record from database
	Fetch(context.Context, string, int) ([]*BotData, error)
}
