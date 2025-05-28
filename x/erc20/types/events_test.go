package types

import (
	"testing"

	"github.com/evmos/evmos/v20/contracts"
)

func TestERC20Events(t *testing.T) {
	if len(contracts.ERC20MinterBurnerDecimalsContract.ABI.Events) <= 0 {
		t.Fatal("expected non-empty abi events")
	}
	if len(contracts.ERC20MinterBurnerDecimalsContract.ABI.Methods) <= 0 {
		t.Fatal("expected non-empty abi methods")
	}

	_, found := contracts.ERC20MinterBurnerDecimalsContract.ABI.Events[EventTypeTransfer]
	if !found {
		t.Fatal("expected event type transfer to be present")
	}

	_, found = contracts.ERC20MinterBurnerDecimalsContract.ABI.Events[EventTypeApproval]
	if !found {
		t.Fatal("expected event type approval to be present")
	}

}
