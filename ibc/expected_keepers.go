package ibc

import (
	libs "github.com/cometbft/cometbft/libs/bytes"
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
)

// DenomTraceKeeper is the reduced interface expectation to instantiate
// a new ERC-20 precompile.
type DenomTraceKeeper interface {
	GetDenomTrace(ctx sdk.Context, hash libs.HexBytes) (transfertypes.DenomTrace, bool)
}
