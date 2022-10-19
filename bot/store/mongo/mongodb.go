package mongo

import (
	"context"
	"log"
	"os"

	"github.com/globalsign/mgo/bson"
	"github.com/oblessing/artisgo/bot/store"
	"github.com/oblessing/artisgo/common"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	db         = "tradingBot"
	collection = "candlesticks"

	logPrefix = "mongo:\t"
)

type MongoDBService struct {
	collection *mongo.Collection
	log        *log.Logger
}

type Config struct {
	url, db, collection string
}

// save data
func (m *MongoDBService) Save(ctx context.Context, data *store.BotData) error {
	_, err := m.collection.InsertOne(ctx, data)
	return err
}

// fetch a specific no of data
func (m *MongoDBService) Fetch(ctx context.Context, pair string, size int) ([]*store.BotData, error) {
	var data []*store.BotData
	r, err := m.collection.Find(ctx, bson.M{"pair": pair}, options.Find().SetSort(bson.M{"date": -1}).SetLimit(int64(size)))
	if err != nil {
		return data, err
	}

	err = r.All(ctx, &data)
	return data, err
}

func startMongoService(config *Config) (*mongo.Collection, error) {
	session, err := common.NewDriver(config.url)
	if err != nil {
		return nil, err
	}
	return session.Database(config.db).Collection(config.collection), nil
}

func NewMongoInstance(url string) (store.Database, error) {
	logger := log.New(os.Stdout, logPrefix, log.LstdFlags|log.Lshortfile)
	return NewMongoInstanceWithLogger(logger, &Config{
		url:        url,
		db:         db,
		collection: collection,
	})
}

func NewMongoInstanceWithLogger(logger *log.Logger, config *Config) (store.Database, error) {
	service, err := startMongoService(config)
	if err != nil {
		return nil, err
	}

	return &MongoDBService{service, logger}, nil
}
