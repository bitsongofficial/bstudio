package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	DefaultCodespace sdk.CodespaceType = "bitsong-media-server"

	CodeInvalidFromAddress sdk.CodeType = 0
	CodeInvalidFileHash    sdk.CodeType = 1
	CodeInvalid            sdk.CodeType = 2

	TypeMsgUpload    = "upload"
	TypeMsgEditTrack = "edit_track"
)

var _ sdk.Msg = MsgUpload{}

type MsgUpload struct {
	FromAddress sdk.AccAddress `json:"from_address"`
	FileHash    string         `json:"file_hash"`
}

func (msg MsgUpload) Route() string { return TypeMsgUpload }
func (msg MsgUpload) Type() string  { return TypeMsgUpload }
func (msg MsgUpload) ValidateBasic() sdk.Error {
	if msg.FromAddress.Empty() {
		return sdk.NewError(DefaultCodespace, CodeInvalidFromAddress, "from address cannot be blank")
	}

	if msg.FileHash == "" {
		return sdk.NewError(DefaultCodespace, CodeInvalidFileHash, "file hash cannot be blank")
	}

	return nil
}

func (msg MsgUpload) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgUpload) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.FromAddress}
}

var _ sdk.Msg = MsgEditTrack{}

type MsgEditTrack struct {
	FromAddress sdk.AccAddress `json:"from_address"`
	TrackId     string         `json:"track_id"`
	Title       string         `json:"title"`
}

func (msg MsgEditTrack) Route() string { return TypeMsgUpload }
func (msg MsgEditTrack) Type() string  { return TypeMsgUpload }
func (msg MsgEditTrack) ValidateBasic() sdk.Error {
	if msg.FromAddress.Empty() {
		return sdk.NewError(DefaultCodespace, CodeInvalidFromAddress, "from address cannot be blank")
	}

	if msg.TrackId == "" {
		return sdk.NewError(DefaultCodespace, CodeInvalid, "track id cannot be blank")
	}

	if msg.Title == "" {
		return sdk.NewError(DefaultCodespace, CodeInvalid, "title cannot be blank")
	}

	return nil
}

func (msg MsgEditTrack) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgEditTrack) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.FromAddress}
}
