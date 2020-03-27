package models

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTrack(t *testing.T) {
	owner := sdk.AccAddress([]byte("owner"))
	duation := float32(20)
	copyright_text := "test copyright"
	copyright_text2 := "test copyright2"

	// Create Track
	track := NewTrack(owner.String(), duation)
	track.Copyright = copyright_text

	err := track.Insert()
	require.NoError(t, err)

	// Get Track
	track2, err := GetTrack(track.ID)
	require.NoError(t, err)
	require.Equal(t, track2.Copyright, copyright_text)

	// Update Track
	track3, err := GetTrack(track.ID)
	require.NoError(t, err)
	track3.Copyright = copyright_text2
	err = track3.Update()
	require.NoError(t, err)
	track3, err = GetTrack(track3.ID)
	require.NoError(t, err)
	require.Equal(t, track3.Copyright, copyright_text2)

	// Delete Track
	track4, err := GetTrack(track3.ID)
	require.NoError(t, err)
	err = track4.Delete()
	require.NoError(t, err)

	_, err = GetTrack(track3.ID)
	require.Error(t, err)
}
