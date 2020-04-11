package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	DefaultCodespace sdk.CodespaceType = "bitsong-media-server"

	CodeInvalidFromAddress sdk.CodeType = 0
	CodeInvalidFileHash    sdk.CodeType = 1
	CodeInvalid            sdk.CodeType = 2
)
