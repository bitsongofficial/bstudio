package models

import (
	"context"
	"fmt"
	"github.com/bitsongofficial/bitsong-media-server/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

const TrackCollection = "track"

type Track struct {
	ID                   primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Title                string             `json:"title" bson:"title"`
	Artists              string             `json:"artists" bson:"artists"`
	Featurings           string             `json:"featurings" bson:"featurings"`
	Producers            string             `json:"producers" bson:"producers"`
	Genre                string             `json:"genre" bson:"genre"`
	Mood                 string             `json:"mood" bson:"mood"`
	ReleaseDate          string             `json:"release_date" bson:"release_date"`
	ReleaseDatePrecision string             `json:"release_date_precision" bson:"release_date_precision"`
	Tags                 string             `json:"tags" bson:"tags"`
	Explicit             bool               `json:"explicit" bson:"explicit"`
	Label                string             `json:"label" bson:"label"`
	Isrc                 string             `json:"isrc" bson:"isrc"`
	UpcEan               string             `json:"upc_ean" bson:"upc_ean"`
	Iswc                 string             `json:"iswc" bson:"iswc"`
	Credits              string             `json:"credits" bson:"credits"`
	Copyright            string             `json:"copyright" bson:"copyright"`   // RR/CC
	Visibility           string             `json:"visibility" bson:"visibility"` // public/private
	Owner                string             `json:"owner" bson:"owner"`
	IsDraft              bool               `json:"is_draft" bson:"is_draft"`
	Audio                string             `json:"audio" bson:"audio"`
	Image                string             `json:"image" bson:"image"`
	Duration             float32            `json:"duration" bson:"duration"`
	CreatedAt            time.Time          `json:"created_at" bson:"created_at"`
}

func NewTrack(owner string, duration float32) *Track {
	return &Track{
		ID:        primitive.NewObjectID(),
		Owner:     owner,
		IsDraft:   true,
		Duration:  duration,
		CreatedAt: time.Now(),
	}
}

func (t *Track) GetCollection() *mongo.Collection {
	db, _ := db.Connect()
	return db.Collection(TrackCollection)
}

func (t *Track) Create() error {
	collection := t.GetCollection()

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	_, err := collection.InsertOne(ctx, t)
	if err != nil {
		return fmt.Errorf("cannot create mongo/track")
	}

	return nil
}

func (t *Transcoder) UpdateAudioHash(cid string) error {
	collection := t.GetCollection()
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

	filter := bson.D{
		{"_id", t.ID},
	}

	update := bson.D{
		{"$set", bson.D{
			{"audio", cid},
		}},
	}

	_, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	return nil
}
