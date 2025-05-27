// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package types

import (
	"errors"
	"math/big"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/evmos/evmos/v20/contracts"
	precompilecommon "github.com/evmos/evmos/v20/precompiles/common"
)

// erc20 events
const (
	EventTypeConvertERC20           = "convert_erc20"
	EventTypeRegisterERC20          = "register_erc20"
	EventTypeToggleTokenConversion  = "toggle_token_conversion" // #nosec
	EventTypeRegisterERC20Extension = "register_erc20_extension"
	EventTypeApproval               = "Approval"
	EventTypeTransfer               = "Transfer"

	AttributeCoinSourceChannel = "source_channel"
	AttributeKeyCosmosCoin     = "cosmos_coin"
	AttributeKeyERC20Token     = "erc20_token" // #nosec
	AttributeKeyReceiver       = "receiver"
)

var (
	ErrEventNotFound = errors.New("event not found in contract abi")
)

// TODO: if we're going with this we should refactor throughout the codebase, where similar code is repeated, e.g. precompiles
func BuildApprovalLog(ctx sdk.Context, erc20Addr, owner, spender common.Address, value *big.Int) (*ethtypes.Log, error) {
	event, found := contracts.ERC20EventsContract.ABI.Events[EventTypeApproval]
	if !found {
		return nil, errorsmod.Wrap(ErrEventNotFound, EventTypeApproval)
	}

	return buildLog(ctx, event, erc20Addr, owner, spender, value)
}

func BuildTransferLog(ctx sdk.Context, erc20Addr, from, to common.Address, value *big.Int) (*ethtypes.Log, error) {
	event, found := contracts.ERC20EventsContract.ABI.Events[EventTypeTransfer]
	if !found {
		return nil, errorsmod.Wrap(ErrEventNotFound, EventTypeTransfer)
	}

	return buildLog(ctx, event, erc20Addr, from, to, value)
}

func buildLog(
	ctx sdk.Context,
	event abi.Event,
	erc20Addr,
	from,
	to common.Address,
	value *big.Int,
) (*ethtypes.Log, error) {
	topics := make([]common.Hash, 3)

	// The first topic is always the signature of the event.
	topics[0] = event.ID

	var err error
	topics[1], err = precompilecommon.MakeTopic(from)
	if err != nil {
		return nil, err
	}

	topics[2], err = precompilecommon.MakeTopic(to)
	if err != nil {
		return nil, err
	}

	arguments := abi.Arguments{event.Inputs[2]}
	packed, err := arguments.Pack(value)
	if err != nil {
		return nil, err
	}

	return &ethtypes.Log{
		Address:     erc20Addr,
		Topics:      topics,
		Data:        packed,
		BlockNumber: uint64(ctx.BlockHeight()), //nolint:gosec // G115
	}, nil
}
