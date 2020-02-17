package db

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

var db *mongo.Database

func Connect() (*mongo.Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017/bitsong-ms?replicaSet=replica01")

	client, err := mongo.NewClient(clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongodb")
	}

	if err := client.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to mongodb")
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping mongodb")
	}

	db = client.Database("bitsong-ms")

	return db, nil
}
