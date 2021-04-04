package mongo

import (
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/oblessing/artisgo/bot/store"
	"log"
	"os"
)

const (
	db         = "tradingBot"
	collection = "candlesticks"

	logPrefix = "mongo:\t"
)

type MongoDBService struct {
	collection *mgo.Collection
	log        *log.Logger
}

type Config struct {
	url, db, collection string
}

// save data
func (m MongoDBService) Save(data *store.BotData) error {
	return m.collection.Insert(data)
}

// fetch a specific no of data
func (m MongoDBService) Fetch(pair string, size int) ([]*store.BotData, error) {
	data := []*store.BotData{}
	err := m.collection.Find(bson.M{"pair": pair}).Sort("-date").Limit(size).All(&data)

	return data, err
}

func startMongoService(config *Config) (*mgo.Collection, error) {
	session, err := mgo.Dial(config.url)
	if err != nil {
		return nil, err
	}
	return session.DB(config.db).C(config.collection), nil
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
	var mongodb interface{} = MongoDBService{service, logger}
	return mongodb.(store.Database), nil
}
