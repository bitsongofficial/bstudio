package db_test

import (
	"github.com/bitsongofficial/bitsong-media-server/db"
	"github.com/bitsongofficial/bitsong-media-server/models"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"testing"
)

func TestConnect(t *testing.T) {
	_, err := db.Connect()
	require.NoError(t, err)
}

func TestCreate(t *testing.T) {
	transcoder := models.NewTranscoder()
	err := transcoder.Create()
	require.NoError(t, err)

	err = transcoder.Delete()
	require.NoError(t, err)
}

func TestGet(t *testing.T) {
	transcoder := models.NewTranscoder()
	err := transcoder.Create()
	require.NoError(t, err)

	t2 := &models.Transcoder{
		ID: transcoder.ID,
	}

	res, err := t2.Get()
	require.NoError(t, err)
	require.Equal(t, res.ID, t2.ID)
	require.NotEqual(t, res.ID, primitive.ObjectID{20})

	err = t2.Delete()
	require.NoError(t, err)
}

func TestUpdatePercentage(t *testing.T) {
	transcoder := models.NewTranscoder()
	err := transcoder.Create()
	require.NoError(t, err)

	err = transcoder.UpdatePercentage(10)
	require.NoError(t, err)

	err = transcoder.Delete()
	require.NoError(t, err)
}

func TestDelete(t *testing.T) {
	transcoder := models.NewTranscoder()
	err := transcoder.Create()
	require.NoError(t, err)

	t2 := &models.Transcoder{
		ID: transcoder.ID,
	}

	_, err = t2.Get()
	require.NoError(t, err)

	err = t2.Delete()
	require.NoError(t, err)
}
