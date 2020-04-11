package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const TypeMsgGetTrack = "get_track"

var _ sdk.Msg = MsgGetTrack{}

type MsgGetTrack struct {
	FromAddress sdk.AccAddress `json:"from_address"`
	TrackId     string         `json:"track_id"`
}

func (msg MsgGetTrack) Route() string { return TypeMsgGetTrack }
func (msg MsgGetTrack) Type() string  { return TypeMsgGetTrack }
func (msg MsgGetTrack) ValidateBasic() sdk.Error {
	if msg.FromAddress.Empty() {
		return sdk.NewError(DefaultCodespace, CodeInvalidFromAddress, "from address cannot be blank")
	}

	if msg.TrackId == "" {
		return sdk.NewError(DefaultCodespace, CodeInvalidFromAddress, "track id cannot be blank")
	}

	return nil
}

func (msg MsgGetTrack) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgGetTrack) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.FromAddress}
}
