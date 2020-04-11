package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const TypeMsgGetTracks = "get_tracks"

var _ sdk.Msg = MsgGetTracks{}

type MsgGetTracks struct {
	FromAddress sdk.AccAddress `json:"from_address"`
}

func (msg MsgGetTracks) Route() string { return TypeMsgGetTracks }
func (msg MsgGetTracks) Type() string  { return TypeMsgGetTracks }
func (msg MsgGetTracks) ValidateBasic() sdk.Error {
	if msg.FromAddress.Empty() {
		return sdk.NewError(DefaultCodespace, CodeInvalidFromAddress, "from address cannot be blank")
	}

	return nil
}

func (msg MsgGetTracks) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgGetTracks) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{msg.FromAddress}
}
