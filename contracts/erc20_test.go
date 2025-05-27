package contracts

import "testing"

func TestERC20Events(t *testing.T) {
	if len(ERC20EventsContract.ABI.Events) <= 0 {
		t.Fatal("expected non-empty abi events")
	}
	if len(ERC20EventsContract.ABI.Methods) <= 0 {
		t.Fatal("expected non-empty abi methods")
	}
}
