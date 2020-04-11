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
	Featurings           string             `json:"featurings,omitempty" bson:"featurings,omitempty"`
	Producers            string             `json:"producers,omitempty" bson:"producers,omitempty"`
	Genre                string             `json:"genre" bson:"genre"`
	Mood                 string             `json:"mood" bson:"mood"`
	ReleaseDate          string             `json:"release_date" bson:"release_date"`
	ReleaseDatePrecision string             `json:"release_date_precision" bson:"release_date_precision"`
	Tags                 string             `json:"tags,omitempty" bson:"tags,omitempty"`
	Explicit             bool               `json:"explicit" bson:"explicit"`
	Label                string             `json:"label,omitempty" bson:"label,omitempty"`
	Isrc                 string             `json:"isrc,omitempty" bson:"isrc,omitempty"`
	UpcEan               string             `json:"upc_ean,omitempty" bson:"upc_ean,omitempty"`
	Iswc                 string             `json:"iswc,omitempty" bson:"iswc,omitempty"`
	Credits              string             `json:"credits,omitempty" bson:"credits,omitempty"`
	Copyright            string             `json:"copyright" bson:"copyright"`   // RR/CC
	Visibility           string             `json:"visibility" bson:"visibility"` // public/private
	Owner                string             `json:"owner" bson:"owner"`
	IsDraft              bool               `json:"is_draft" bson:"is_draft"`
	Audio                string             `json:"audio" bson:"audio"`
	AudioOriginal        string             `json:"audio_original" bson:"audio_original"`
	Image                string             `json:"image" bson:"image"`
	Duration             float32            `json:"duration" bson:"duration"`
	RewardsUsers         string             `json:"rewards_users" bson:"rewards_users"`
	RewardsPlaylists     string             `json:"rewards_playlists" bson:"rewards_playlists"`
	RightsHolders        string             `json:"rights_holders" bson:"rights_holders"`
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
	if t.Title != "" && t.Artists != "" && t.Genre != "" && t.Mood != "" && t.ReleaseDate != "" && t.ReleaseDatePrecision != "" && t.Copyright != "" && t.Visibility != "" && t.Audio != "" && t.Image != "" && t.Duration > 0 && t.RightsHolders != "" && t.RewardsUsers != "" && t.RewardsPlaylists != "" {
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

	fields := bson.D{
		bson.E{Key: "title", Value: t.Title},
		bson.E{Key: "artists", Value: t.Artists},
		bson.E{Key: "featurings", Value: t.Featurings},
		bson.E{Key: "producers", Value: t.Producers},
		bson.E{Key: "genre", Value: t.Genre},
		bson.E{Key: "mood", Value: t.Mood},
		bson.E{Key: "release_date", Value: t.ReleaseDate},
		bson.E{Key: "release_date_precision", Value: t.ReleaseDatePrecision},
		bson.E{Key: "tags", Value: t.Tags},
		bson.E{Key: "label", Value: t.Label},
		bson.E{Key: "isrc", Value: t.Isrc},
		bson.E{Key: "upc_ean", Value: t.UpcEan},
		bson.E{Key: "iswc", Value: t.Iswc},
		bson.E{Key: "credits", Value: t.Credits},
		bson.E{Key: "copyright", Value: t.Copyright},
		bson.E{Key: "visibility", Value: t.Visibility},
		bson.E{Key: "audio", Value: t.Audio},
		bson.E{Key: "audio_original", Value: t.AudioOriginal},
		bson.E{Key: "image", Value: t.Image},
		bson.E{Key: "duration", Value: t.Duration},
		bson.E{Key: "explicit", Value: t.Explicit},
		bson.E{Key: "is_draft", Value: !t.IsCompleted()},
		bson.E{Key: "rights_holders", Value: t.RightsHolders},
		bson.E{Key: "rewards_users", Value: t.RewardsUsers},
		bson.E{Key: "rewards_playlists", Value: t.RewardsPlaylists},
	}

	update := bson.D{
		{"$set", fields},
	}

	if _, err := collection.UpdateOne(ctx, filter, update); err != nil {
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
