package backend

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/ethereum/go-ethereum/common/math"
	rpctypes "github.com/evmos/evmos/v19/rpc/types"
	evmtypes "github.com/evmos/evmos/v19/x/evm/types"
	feemarkettypes "github.com/evmos/evmos/v19/x/feemarket/types"
)

var baseFeeDeltaBlocks = big.NewInt(2)

func (b *Backend) calculateFeePayerFees(gas uint64) (amount sdkmath.Int, err error) {
	// Get current base fee
	var baseFee *big.Int
	blockRes, err := b.TendermintBlockResultByNumber(nil)
	if err != nil {
		err = fmt.Errorf("failed to query latest block: %w", err)
		return
	}
	res, err := b.queryClient.BaseFee(rpctypes.ContextWithHeight(blockRes.Height), &evmtypes.QueryBaseFeeRequest{})
	if err != nil || res.BaseFee == nil {
		err = fmt.Errorf("failed to query base fee: %w", err)
		return
	}
	if res.BaseFee.Sign() == 0 {
		sdkmath.NewInt(0)
		return
	}
	baseFee = res.BaseFee.BigInt()

	// Get chain params
	params, err := b.queryClient.FeeMarket.Params(b.ctx, &feemarkettypes.QueryParamsRequest{})
	if err != nil {
		err = fmt.Errorf("failed to query params: %w", err)
		return
	}

	// Adjust to cover maximum increase of base fee in `baseFeeDeltaBlocks` blocks
	// (X(a+1)^b)/a^b where
	//   X is the original base fee
	//   a is the base fee change denominator
	//   b is `baseFeeDeltaBlocks`
	baseFeeChangeDenominator := big.NewInt(int64(params.Params.BaseFeeChangeDenominator))
	d := new(big.Int).Exp(baseFeeChangeDenominator, baseFeeDeltaBlocks, nil)
	m := new(big.Int).Exp(new(big.Int).Add(baseFeeChangeDenominator, big.NewInt(1)), baseFeeDeltaBlocks, nil)
	newBaseFee := new(big.Int).Div(new(big.Int).Mul(baseFee, m), d)
	baseFee = math.BigMax(
		newBaseFee,
		new(big.Int).Mul(big.NewInt(1), baseFeeDeltaBlocks), // Minimum delta is 1
	)

	gasInt := new(big.Int).SetUint64(gas)
	amount = sdkmath.NewIntFromBigInt(new(big.Int).Mul(baseFee, gasInt))
	return
}

func (b *Backend) feePayerTx(clientCtx client.Context, ethereumMsg *evmtypes.MsgEthereumTx, evmDenom string) (tx authsigning.Tx, err error) {
	if b.feePayerPrivKey == nil {
		panic("no fee payer priv key")
	}
	privKey := *b.feePayerPrivKey
	pubKey := privKey.PubKey()

	// Add the extension options to the transaction for the ethereum message
	txBuilder, ok := clientCtx.TxConfig.NewTxBuilder().(authtx.ExtensionOptionsTxBuilder)
	if !ok {
		err = fmt.Errorf("unsupported builder: %T", b)
		return
	}
	option, err := codectypes.NewAnyWithValue(&evmtypes.ExtensionOptionsEthereumTx{})
	if err != nil {
		return
	}
	txBuilder.SetExtensionOptions(option)

	// Set gas limit from the ethereum message
	gas := ethereumMsg.GetGas()
	txBuilder.SetGasLimit(gas)

	// Overwrite user-provided fees
	feeAmt, err := b.calculateFeePayerFees(gas)
	if err != nil {
		return
	}
	fees := make(sdk.Coins, 0, 1)
	if feeAmt.Sign() > 0 {
		fees = append(fees, sdk.NewCoin(evmDenom, feeAmt))
	}
	txBuilder.SetFeeAmount(fees)

	// A valid msg should have empty From field
	ethereumMsg.From = ""

	// Set message in the transaction
	err = txBuilder.SetMsgs(ethereumMsg)
	if err != nil {
		return
	}

	// Add the fee payer information
	feepayerAddress := sdk.AccAddress(pubKey.Address())
	txBuilder.SetFeePayer(feepayerAddress)

	// Query the account number and sequence from the remote chain
	accountNumber, sequence, err := clientCtx.AccountRetriever.GetAccountNumberSequence(clientCtx, feepayerAddress)
	if err != nil {
		err = fmt.Errorf("failed to get account: %w", err)
		return
	}

	// Make sure AuthInfo is complete before signing
	sigData := signing.SingleSignatureData{
		SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
		Signature: nil,
	}
	sigV2 := signing.SignatureV2{
		PubKey:   pubKey,
		Data:     &sigData,
		Sequence: sequence,
	}
	err = txBuilder.SetSignatures(sigV2)
	if err != nil {
		return
	}

	// Sign and set signatures
	signerData := authsigning.SignerData{
		ChainID:       clientCtx.ChainID,
		AccountNumber: accountNumber,
		Sequence:      sequence,
	}
	sig, err := clienttx.SignWithPrivKey(
		signing.SignMode_SIGN_MODE_DIRECT,
		signerData,
		txBuilder,
		&privKey,
		clientCtx.TxConfig,
		sequence,
	)
	if err != nil {
		err = fmt.Errorf("failed to sign transaction: %w", err)
		return
	}
	err = txBuilder.SetSignatures(sig)
	if err != nil {
		err = fmt.Errorf("failed to set signatures: %w", err)
		return
	}

	tx = txBuilder.GetTx()
	return
}
