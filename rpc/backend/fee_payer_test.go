package backend

import (
	"context"
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	tmrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/evmos/evmos/v19/rpc/backend/mocks"
	evmtypes "github.com/evmos/evmos/v19/x/evm/types"
)

func (suite *BackendTestSuite) feePayerTxBytes(ethTx *evmtypes.MsgEthereumTx, sequence uint64) types.Tx {
	cosmosTx, _ := suite.backend.feePayer.buildTx(ethTx, evmtypes.DefaultEVMDenom, 2, sequence)
	txBytes, _ := suite.backend.clientCtx.TxConfig.TxEncoder()(cosmosTx)
	return txBytes
}

func (suite *BackendTestSuite) TestSendRawTransactionFeePayer() {
	return //TODO
	ethTx, bz := suite.buildEthereumTx()

	// Sign the ethTx
	queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
	RegisterParamsWithoutHeader(queryClient, 1)
	ethSigner := ethtypes.LatestSigner(suite.backend.ChainConfig())
	err := ethTx.Sign(ethSigner, suite.signer)
	suite.Require().NoError(err)
	rlpEncodedBz, _ := rlp.EncodeToBytes(ethTx.AsTransaction())

	testCases := []struct {
		name         string
		registerMock func()
		rawTx        []byte
		expHash      common.Hash
		expPass      bool
	}{
		{
			"fail - empty bytes",
			func() {
			},
			[]byte{},
			common.Hash{},
			false,
		},
		{
			"fail - no RLP encoded bytes",
			func() {
			},
			bz,
			common.Hash{},
			false,
		},
		{
			"fail - unprotected transactions",
			func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParamsWithoutHeaderError(queryClient, 1)
			},
			rlpEncodedBz,
			common.Hash{},
			false,
		},
		{
			"fail - failed to get evm params",
			func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParamsWithoutHeaderError(queryClient, 1)
			},
			rlpEncodedBz,
			common.Hash{},
			false,
		},
		{
			"fail - failed to broadcast transaction",
			func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParamsWithoutHeader(queryClient, 1)
				RegisterFeeMarketParams(suite.backend.queryClient.FeeMarket.(*mocks.FeeMarketQueryClient), 1)
				RegisterBaseFee(queryClient, sdkmath.NewInt(123))
				RegisterBlockResults(client, 1)

				txBytes := suite.feePayerTxBytes(ethTx, 1)
				RegisterBroadcastTxError(client, txBytes)
			},
			rlpEncodedBz,
			common.HexToHash(ethTx.Hash),
			false,
		},
		{
			"pass - Gets the correct transaction hash of the eth transaction",
			func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParamsWithoutHeader(queryClient, 1)
				RegisterFeeMarketParams(suite.backend.queryClient.FeeMarket.(*mocks.FeeMarketQueryClient), 1)
				RegisterBaseFee(queryClient, sdkmath.NewInt(123))
				RegisterBlockResults(client, 1)

				txBytes := suite.feePayerTxBytes(ethTx, 1)
				RegisterBroadcastTx(client, txBytes)
			},
			rlpEncodedBz,
			common.HexToHash(ethTx.Hash),
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("case %s", tc.name), func() {
			suite.SetupTest("00") // reset test and queries
			tc.registerMock()

			hash, err := suite.backend.SendRawTransaction(tc.rawTx)

			if tc.expPass {
				suite.Require().Equal(tc.expHash, hash)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) buildEthereumTxNonce(nonce uint64) *evmtypes.MsgEthereumTx {
	ethTxParams := evmtypes.EvmTxArgs{
		ChainID:  suite.backend.chainID,
		Nonce:    nonce,
		To:       &common.Address{},
		Amount:   big.NewInt(0),
		GasLimit: 100000,
		GasPrice: big.NewInt(0),
	}
	msgEthereumTx := evmtypes.NewTx(&ethTxParams)

	// A valid msg should have empty `From`
	msgEthereumTx.From = suite.from.Hex()

	txBuilder := suite.backend.clientCtx.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msgEthereumTx)
	suite.Require().NoError(err)

	return msgEthereumTx
}
func (suite *BackendTestSuite) TestSendRawTransactionFeePayerSequence() {
	// Reset test env
	suite.SetupTest("00")
	ctxClient := suite.backend.clientCtx.Client.(*mocks.Client)
	queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)

	RegisterParamsWithoutHeader(queryClient, 1)
	RegisterFeeMarketParams(suite.backend.queryClient.FeeMarket.(*mocks.FeeMarketQueryClient), 1)
	RegisterBaseFee(queryClient, sdkmath.NewInt(123))
	RegisterBlockResults(ctxClient, 1)

	tar, _ := suite.backend.clientCtx.AccountRetriever.(client.TestAccountRetriever)
	acc := tar.Accounts[suite.feePayerAcc.String()]

	for i := uint64(1); i < 1000; i++ {
		ethTx := suite.buildEthereumTxNonce(i)
		ethSigner := ethtypes.LatestSigner(suite.backend.ChainConfig())
		err := ethTx.Sign(ethSigner, suite.signer)
		suite.Require().NoError(err)
		rlpEncodedBz, _ := rlp.EncodeToBytes(ethTx.AsTransaction())

		txBytes := suite.feePayerTxBytes(ethTx, acc.Seq)
		if i == 10 {
			ctxClient.On("BroadcastTxSync", context.Background(), txBytes).
				Return(&tmrpctypes.ResultBroadcastTx{
					Code: sdkerrors.ErrWrongSequence.ABCICode(),
				}, nil)

			_, err = suite.backend.SendRawTransaction(rlpEncodedBz)
			suite.Require().Error(err)
			continue
		}

		RegisterBroadcastTx(ctxClient, txBytes)
		_, err = suite.backend.SendRawTransaction(rlpEncodedBz)
		suite.Require().NoError(err)

		// Increment account sequence in the account retriever
		acc.Seq++
		tar.Accounts[suite.feePayerAcc.String()] = acc
	}
}
