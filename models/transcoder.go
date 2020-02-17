package models

import (
	"context"
	"fmt"
	"github.com/angelorc/go-uploader/db"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

const Collection = "transcoder"

type Transcoder struct {
	ID          primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Percentage  int                `json:"percentage" bson:"percentage"`
	UploadID    uuid.UUID          `json:"upload_id" bson:"upload_id"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	CompletedAt time.Time          `json:"completed_at" bson:"completed_at"`
}

func NewTranscoder(uid uuid.UUID) *Transcoder {
	return &Transcoder{
		ID:         primitive.NewObjectID(),
		Percentage: 0,
		UploadID:   uid,
		CreatedAt:  time.Now(),
	}
}

func (t *Transcoder) GetCollection() *mongo.Collection {
	db, _ := db.Connect()

	return db.Collection(Collection)
}

func (t *Transcoder) Create() error {
	collection := t.GetCollection()

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	_, err := collection.InsertOne(ctx, t)
	if err != nil {
		return fmt.Errorf("cannot create mongo/transcoder")
	}

	return nil
}

func (t *Transcoder) Get() (*Transcoder, error) {
	collection := t.GetCollection()
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

	filter := bson.D{
		{"_id", t.ID},
	}

	var transcoder Transcoder
	err := collection.FindOne(ctx, filter).Decode(&transcoder)
	if err != nil {
		return nil, err
	}

	return &transcoder, nil
}

func (t *Transcoder) UpdatePercentage(percentage int) error {
	collection := t.GetCollection()
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

	filter := bson.D{
		{"_id", t.ID},
	}

	update := bson.D{
		{"$set", bson.D{
			{"percentage", percentage},
		}},
	}

	if percentage == 100 {
		update = bson.D{
			{"$set", bson.D{
				{"percentage", percentage},
				{"completed_at", time.Now()},
			}},
		}
	}

	_, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	return nil
}

func (t *Transcoder) Delete() error {
	collection := t.GetCollection()
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

	filter := bson.D{
		{"_id", t.ID},
	}

	_, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	return nil
}
