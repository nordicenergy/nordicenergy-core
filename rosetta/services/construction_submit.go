package services

import (
	"context"
	"fmt"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/pkg/errors"

	ngyTypes "github.com/nordicenergy/nordicenergy-core/core/types"
	"github.com/nordicenergy/nordicenergy-core/rosetta/common"
	stakingTypes "github.com/nordicenergy/nordicenergy-core/staking/types"
)

// ConstructionHash implements the /construction/hash endpoint.
func (s *ConstructAPI) ConstructionHash(
	ctx context.Context, request *types.ConstructionHashRequest,
) (*types.TransactionIdentifierResponse, *types.Error) {
	if err := assertValidNetworkIdentifier(request.NetworkIdentifier, s.ngy.ShardID); err != nil {
		return nil, err
	}
	_, tx, rosettaError := unpackWrappedTransactionFromString(request.SignedTransaction)
	if rosettaError != nil {
		return nil, rosettaError
	}
	if tx == nil {
		return nil, common.NewError(common.CatchAllError, map[string]interface{}{
			"message": "nil transaction",
		})
	}
	if tx.ShardID() != s.ngy.ShardID {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": fmt.Sprintf("transaction is for shard %v != shard %v", tx.ShardID(), s.ngy.ShardID),
		})
	}
	return &types.TransactionIdentifierResponse{
		TransactionIdentifier: &types.TransactionIdentifier{Hash: tx.Hash().String()},
	}, nil
}

// ConstructionSubmit implements the /construction/submit endpoint.
func (s *ConstructAPI) ConstructionSubmit(
	ctx context.Context, request *types.ConstructionSubmitRequest,
) (*types.TransactionIdentifierResponse, *types.Error) {
	if err := assertValidNetworkIdentifier(request.NetworkIdentifier, s.ngy.ShardID); err != nil {
		return nil, err
	}
	wrappedTransaction, tx, rosettaError := unpackWrappedTransactionFromString(request.SignedTransaction)
	if rosettaError != nil {
		return nil, rosettaError
	}
	if wrappedTransaction == nil || tx == nil {
		return nil, common.NewError(common.CatchAllError, map[string]interface{}{
			"message": "nil wrapped transaction or nil unwrapped transaction",
		})
	}
	if tx.ShardID() != s.ngy.ShardID {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": fmt.Sprintf("transaction is for shard %v != shard %v", tx.ShardID(), s.ngy.ShardID),
		})
	}

	wrappedSenderAddress, err := getAddress(wrappedTransaction.From)
	if err != nil {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": errors.WithMessage(err, "unable to get address from wrapped transaction"),
		})
	}
	txSenderAddress, err := tx.SenderAddress()
	if err != nil {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": errors.WithMessage(err, "unable to get sender address from transaction").Error(),
		})
	}
	if wrappedSenderAddress != txSenderAddress {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": "transaction sender address does not match wrapped transaction sender address",
		})
	}

	if stakingTx, ok := tx.(*stakingTypes.StakingTransaction); ok && wrappedTransaction.IsStaking {
		if err := s.ngy.SendStakingTx(ctx, stakingTx); err != nil {
			return nil, common.NewError(common.StakingTransactionSubmissinetrror, map[string]interface{}{
				"message": err.Error(),
			})
		}
	} else if plainTx, ok := tx.(*ngyTypes.Transaction); ok && !wrappedTransaction.IsStaking {
		if err := s.ngy.SendTx(ctx, plainTx); err != nil {
			return nil, common.NewError(common.TransactionSubmissinetrror, map[string]interface{}{
				"message": err.Error(),
			})
		}
	} else {
		return nil, common.NewError(common.CatchAllError, map[string]interface{}{
			"message": "invalid/inconsistent type or unknown transaction type stored in wrapped transaction",
		})
	}

	return &types.TransactionIdentifierResponse{
		TransactionIdentifier: &types.TransactionIdentifier{Hash: tx.Hash().String()},
	}, nil
}
