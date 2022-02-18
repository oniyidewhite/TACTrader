package common

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

const name = "mongo"

type Config struct {
	URI     string
	Timeout int
}

func NewDriver(uri string) (*mongo.Client, error) {
	if len(uri) == 0 {
		return nil, errors.New("invalid_mongo_uri")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10)*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	return client, nil
}
