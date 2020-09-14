package database

import (
	"context"
	"errors"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"time"
)

var (
	UploadCollection *mongo.Collection
)

type Database struct {
	Url              string
	Name             string
	UploadCollection *mongo.Collection
}

func NewDatabase(url, name string) *Database {
	return &Database{
		Url:              url,
		Name:             name,
		UploadCollection: nil,
	}
}

func (db *Database) Init() (error, context.CancelFunc) {
	client, _ := mongo.NewClient(options.Client().ApplyURI(db.Url))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	_ = client.Connect(ctx)

	pingErr := client.Ping(context.Background(), readpref.Primary())
	if pingErr != nil {
		return pingErr, cancel
	}

	log.Info().Str("mongodb", db.Name).Msg("mongodb connected...")

	mdb := client.Database(db.Name)
	db.UploadCollection = mdb.Collection("uploads")

	return nil, cancel
}

func (db *Database) FindOne(collection *mongo.Collection, filter bson.M) (bson.M, error) {
	// Creates a document
	var model bson.M

	// Finds a model in the database and handles any possible errors
	// Note that it returns `nil` if model has been found
	err := collection.FindOne(context.Background(), filter).Decode(&model)
	if err != nil {
		return nil, err
	}

	return model, nil
}

func (db *Database) InsertOne(collection *mongo.Collection, model interface{}) (primitive.ObjectID, error) {
	id, err := collection.InsertOne(context.Background(), model)
	if err != nil {
		return primitive.NilObjectID, err
	}

	return id.InsertedID.(primitive.ObjectID), nil
}

func (db *Database) UpdateOne(collection *mongo.Collection, id primitive.ObjectID, update bson.M) error {
	res, err := collection.UpdateOne(context.Background(), bson.M{"_id": id}, bson.M{"$set": update})
	if err != nil || res.ModifiedCount == 0 {
		return errors.New("mongo: no documents in result")
	}

	return nil
}

func (db *Database) DecodeStruct(s interface{}) (bson.M, error) {
	// Creating a bson.M variable
	var decoded bson.M

	// Marshalling the input struct
	encoded, err := bson.Marshal(s)
	if err != nil {
		return nil, err
	}

	// Unmarshalling the encoded object
	err = bson.Unmarshal(encoded, &decoded)
	if err != nil {
		return nil, err
	}

	return decoded, nil
}
