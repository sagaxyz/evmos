package ics20

import (
	"context"

	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
)

// TransferKeeper is the expected restricted interface for the ICS-20 precompile.
type TransferKeeper interface {
	DenomTrace(ctx context.Context, req *transfertypes.QueryDenomTraceRequest) (*transfertypes.QueryDenomTraceResponse, error)
	DenomTraces(ctx context.Context, req *transfertypes.QueryDenomTracesRequest) (*transfertypes.QueryDenomTracesResponse, error)
	DenomHash(ctx context.Context, req *transfertypes.QueryDenomHashRequest) (*transfertypes.QueryDenomHashResponse, error)
	Transfer(ctx context.Context, msg *transfertypes.MsgTransfer) (*transfertypes.MsgTransferResponse, error)
}
