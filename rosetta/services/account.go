package services

import (
	"context"
	"fmt"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"

	ngyTypes "github.com/nordicenergy/nordicenergy-core/core/types"
	"github.com/nordicenergy/nordicenergy-core/ngy"
	internalCommon "github.com/nordicenergy/nordicenergy-core/internal/common"
	"github.com/nordicenergy/nordicenergy-core/rosetta/common"
)

// AccountAPI implements the server.AccountAPIServicer interface.
type AccountAPI struct {
	ngy *ngy.nordicenergy
}

// NewAccountAPI creates a new instance of a BlockAPI.
func NewAccountAPI(ngy *ngy.nordicenergy) server.AccountAPIServicer {
	return &AccountAPI{
		ngy: ngy,
	}
}

// AccountBalance implements the /account/balance endpoint
func (s *AccountAPI) AccountBalance(
	ctx context.Context, request *types.AccountBalanceRequest,
) (*types.AccountBalanceResponse, *types.Error) {
	if err := assertValidNetworkIdentifier(request.NetworkIdentifier, s.ngy.ShardID); err != nil {
		return nil, err
	}

	var block *ngyTypes.Block
	var rosettaError *types.Error
	if request.BlockIdentifier == nil {
		block = s.ngy.CurrentBlock()
	} else {
		block, rosettaError = getBlock(ctx, s.ngy, request.BlockIdentifier)
		if rosettaError != nil {
			return nil, rosettaError
		}
	}

	addr, err := getAddress(request.AccountIdentifier)
	if err != nil {
		return nil, common.NewError(common.SanityCheckError, map[string]interface{}{
			"message": err.Error(),
		})
	}
	blockNum := rpc.BlockNumber(block.Header().Header.Number().Int64())
	balance, err := s.ngy.GetBalance(ctx, addr, blockNum)
	if err != nil {
		return nil, common.NewError(common.SanityCheckError, map[string]interface{}{
			"message": "invalid address",
		})
	}

	amount := types.Amount{
		Value:    balance.String(),
		Currency: &common.NativeCurrency,
	}

	respBlock := types.BlockIdentifier{
		Index: blockNum.Int64(),
		Hash:  block.Header().Hash().String(),
	}

	return &types.AccountBalanceResponse{
		BlockIdentifier: &respBlock,
		Balances:        []*types.Amount{&amount},
	}, nil
}

// AccountMetadata used for account identifiers
type AccountMetadata struct {
	Address string `json:"hex_address"`
}

// newAccountIdentifier ..
func newAccountIdentifier(
	address ethCommon.Address,
) (*types.AccountIdentifier, *types.Error) {
	b32Address, err := internalCommon.AddressToBech32(address)
	if err != nil {
		return nil, common.NewError(common.SanityCheckError, map[string]interface{}{
			"message": err.Error(),
		})
	}
	metadata, err := types.MarshalMap(AccountMetadata{Address: address.String()})
	if err != nil {
		return nil, common.NewError(common.CatchAllError, map[string]interface{}{
			"message": err.Error(),
		})
	}

	return &types.AccountIdentifier{
		Address:  b32Address,
		Metadata: metadata,
	}, nil
}

// getAddress ..
func getAddress(
	identifier *types.AccountIdentifier,
) (ethCommon.Address, error) {
	if identifier == nil {
		return ethCommon.Address{}, fmt.Errorf("identifier cannot be nil")
	}
	return internalCommon.Bech32ToAddress(identifier.Address)
}
