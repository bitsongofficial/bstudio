package types

import "github.com/cosmos/cosmos-sdk/codec"

var ModuleCdc = codec.New()

/*func init() {
	RegisterCodec(ModuleCdc)
}*/

// RegisterCodec registers concrete types on the Amino codec
func RegisterCodec(cdc *codec.Codec) {
	cdc.RegisterConcrete(MsgUpload{}, "bitsong-media-server/MsgUpload", nil)
	cdc.RegisterConcrete(MsgGetTrack{}, "bitsong-media-server/MsgGetTrack", nil)
	cdc.RegisterConcrete(MsgGetTracks{}, "bitsong-media-server/MsgGetTracks", nil)
	cdc.RegisterConcrete(MsgEditTrack{}, "bitsong-media-server/MsgEditTrack", nil)
}
