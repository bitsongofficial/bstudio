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
	Featurings           *string            `json:"featurings,omitempty" bson:"featurings,omitempty"`
	Producers            *string            `json:"producers,omitempty" bson:"producers,omitempty"`
	Genre                string             `json:"genre" bson:"genre"`
	Mood                 string             `json:"mood" bson:"mood"`
	ReleaseDate          string             `json:"release_date" bson:"release_date"`
	ReleaseDatePrecision string             `json:"release_date_precision" bson:"release_date_precision"`
	Tags                 *string            `json:"tags,omitempty" bson:"tags,omitempty"`
	Explicit             bool               `json:"explicit" bson:"explicit"`
	Label                *string            `json:"label,omitempty" bson:"label,omitempty"`
	Isrc                 *string            `json:"isrc,omitempty" bson:"isrc,omitempty"`
	UpcEan               *string            `json:"upc_ean,omitempty" bson:"upc_ean,omitempty"`
	Iswc                 *string            `json:"iswc,omitempty" bson:"iswc,omitempty"`
	Credits              *string            `json:"credits,omitempty" bson:"credits,omitempty"`
	Copyright            string             `json:"copyright" bson:"copyright"`   // RR/CC
	Visibility           string             `json:"visibility" bson:"visibility"` // public/private
	Owner                string             `json:"owner" bson:"owner"`
	IsDraft              bool               `json:"is_draft" bson:"is_draft"`
	Audio                string             `json:"audio" bson:"audio"`
	AudioOriginal        string             `json:"audio_original" bson:"audio_original"`
	Image                string             `json:"image" bson:"image"`
	Duration             float32            `json:"duration" bson:"duration"`
	CreatedAt            time.Time          `json:"created_at" bson:"created_at"`
}

func NewTrack(title, owner string, duration float32) *Track {
	return &Track{
		ID:        primitive.NewObjectID(),
		Title:     title,
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

func GetTracksByOwner(owner string) (*[]Track, error) {
	collection := GetTrackCollection()
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

	filter := bson.D{
		{"owner", owner},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	var tracks []Track
	if err = cursor.All(ctx, &tracks); err != nil {
		return nil, err
	}

	return &tracks, nil
}

func (t *Track) IsCompleted() bool {
	if t.Title != "" && t.Artists != "" && t.Genre != "" && t.Mood != "" && t.ReleaseDate != "" && t.ReleaseDatePrecision != "" && t.Copyright != "" && t.Visibility != "" && t.Audio != "" && t.Image != "" && t.Duration > 0 {
		return true
	}

	return false
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
	if t.Featurings != nil {
		fields = append(fields, bson.E{"featurings", t.Featurings})
	}
	if t.Producers != nil {
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
	if t.Tags != nil {
		fields = append(fields, bson.E{"tags", t.Tags})
	}
	if t.Label != nil {
		fields = append(fields, bson.E{"label", t.Label})
	}
	if t.Isrc != nil {
		fields = append(fields, bson.E{"isrc", t.Isrc})
	}
	if t.UpcEan != nil {
		fields = append(fields, bson.E{"upc_ean", t.UpcEan})
	}
	if t.Iswc != nil {
		fields = append(fields, bson.E{"iswc", t.Iswc})
	}
	if t.Credits != nil {
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
	if t.AudioOriginal != "" {
		fields = append(fields, bson.E{"audio_original", t.AudioOriginal})
	}
	if t.Image != "" {
		fields = append(fields, bson.E{"image", t.Image})
	}
	if t.Duration > 0 {
		fields = append(fields, bson.E{"duration", t.Duration})
	}

	fields = append(fields, bson.E{Key: "explicit", Value: t.Explicit})
	fields = append(fields, bson.E{Key: "is_draft", Value: !t.IsCompleted()})

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
