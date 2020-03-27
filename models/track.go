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

func GetTrackCollection() *mongo.Collection {
	db, _ := db.Connect()
	return db.Collection(TrackCollection)
}

// insert
func (t *Track) Insert() error {
	collection := GetTrackCollection()

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	_, err := collection.InsertOne(ctx, t)
	if err != nil {
		return fmt.Errorf("cannot insert track")
	}

	return nil
}

// get
func GetTrack(trackID primitive.ObjectID) (*Track, error) {
	collection := GetTrackCollection()
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

	filter := bson.D{
		{"_id", trackID},
	}

	var track Track
	err := collection.FindOne(ctx, filter).Decode(&track)
	if err != nil {
		return nil, err
	}

	return &track, nil
}

// update
func (t *Track) Update() error {
	collection := GetTrackCollection()
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

	filter := bson.D{
		{"_id", t.ID},
	}

	var fields bson.D

	if t.Title != "" {
		fields = append(fields, bson.E{"title", t.Title})
	}
	if t.Artists != "" {
		fields = append(fields, bson.E{"artists", t.Artists})
	}
	if t.Featurings != "" {
		fields = append(fields, bson.E{"featurings", t.Featurings})
	}
	if t.Producers != "" {
		fields = append(fields, bson.E{"producers", t.Producers})
	}
	if t.Genre != "" {
		fields = append(fields, bson.E{"genre", t.Genre})
	}
	if t.Mood != "" {
		fields = append(fields, bson.E{"mood", t.Mood})
	}
	if t.ReleaseDate != "" {
		fields = append(fields, bson.E{"release_date", t.ReleaseDate})
	}
	if t.ReleaseDatePrecision != "" {
		fields = append(fields, bson.E{"release_date_precision", t.ReleaseDatePrecision})
	}
	if t.Tags != "" {
		fields = append(fields, bson.E{"tags", t.Tags})
	}
	if t.Label != "" {
		fields = append(fields, bson.E{"label", t.Label})
	}
	if t.Isrc != "" {
		fields = append(fields, bson.E{"isrc", t.Isrc})
	}
	if t.UpcEan != "" {
		fields = append(fields, bson.E{"upc_ean", t.UpcEan})
	}
	if t.Iswc != "" {
		fields = append(fields, bson.E{"iswc", t.Iswc})
	}
	if t.Credits != "" {
		fields = append(fields, bson.E{"credits", t.Credits})
	}
	if t.Copyright != "" {
		fields = append(fields, bson.E{"copyright", t.Copyright})
	}
	if t.Visibility != "" {
		fields = append(fields, bson.E{"visibility", t.Visibility})
	}
	if t.Audio != "" {
		fields = append(fields, bson.E{"audio", t.Audio})
	}
	if t.Image != "" {
		fields = append(fields, bson.E{"image", t.Image})
	}
	if t.Duration != 0 {
		fields = append(fields, bson.E{"duration", t.Duration})
	}
	if t.Explicit {
		fields = append(fields, bson.E{"explicit", t.Explicit})
	}
	if !t.IsDraft {
		fields = append(fields, bson.E{"is_draft", t.IsDraft})
	}

	if len(fields) == 0 {
		return nil
	}

	update := bson.D{
		{"$set", fields},
	}

	_, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	return nil
}

// delete
func (t *Track) Delete() error {
	collection := GetTrackCollection()
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
