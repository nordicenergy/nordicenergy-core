package services

import (
	"encoding/json"
	"fmt"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"

	ngyTypes "github.com/nordicenergy/nordicenergy-core/core/types"
	"github.com/nordicenergy/nordicenergy-core/rosetta/common"
)

// TransactionMetadata contains all (optional) information for a transaction.
type TransactionMetadata struct {
	// CrossShardIdentifier is the transaction identifier on the from/source shard
	CrossShardIdentifier *types.TransactionIdentifier `json:"cross_shard_transaction_identifier,omitempty"`
	ToShardID            *uint32                      `json:"to_shard,omitempty"`
	FromShardID          *uint32                      `json:"from_shard,omitempty"`
	// ContractAccountIdentifier is the 'main' contract account ID associated with a transaction
	ContractAccountIdentifier *types.AccountIdentifier `json:"contract_account_identifier,omitempty"`
	Data                      *string                  `json:"data,omitempty"`
	Logs                      []*ngyTypes.Log          `json:"logs,omitempty"`
}

// UnmarshalFromInterface ..
func (t *TransactionMetadata) UnmarshalFromInterface(metaData interface{}) error {
	var args TransactionMetadata
	dat, err := json.Marshal(metaData)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(dat, &args); err != nil {
		return err
	}
	*t = args
	return nil
}

// ConstructTransaction object (unsigned).
// TODO (dm): implement staking transaction construction
func ConstructTransaction(
	compnetnts *OperationCompnetnts, metadata *ConstructMetadata, sourceShardID uint32,
) (response ngyTypes.PoolTransaction, rosettaError *types.Error) {
	if compnetnts == nil || metadata == nil {
		return nil, common.NewError(common.CatchAllError, map[string]interface{}{
			"message": "nil compnetnts or metadata",
		})
	}
	if metadata.Transaction.FromShardID != nil && *metadata.Transaction.FromShardID != sourceShardID {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": fmt.Sprintf("intended source shard %v != tx metadata source shard %v",
				sourceShardID, *metadata.Transaction.FromShardID),
		})
	}

	var tx ngyTypes.PoolTransaction
	switch compnetnts.Type {
	case common.NativeCrossShardTransferOperation:
		if tx, rosettaError = constructCrossShardTransaction(compnetnts, metadata, sourceShardID); rosettaError != nil {
			return nil, rosettaError
		}
	case common.ContractCreationOperation:
		if tx, rosettaError = constructContractCreationTransaction(compnetnts, metadata, sourceShardID); rosettaError != nil {
			return nil, rosettaError
		}
	case common.NativeTransferOperation:
		if tx, rosettaError = constructPlainTransaction(compnetnts, metadata, sourceShardID); rosettaError != nil {
			return nil, rosettaError
		}
	default:
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": fmt.Sprintf("cannot create transaction with compnetnt type %v", compnetnts.Type),
		})
	}
	return tx, nil
}

// constructCrossShardTransaction ..
func constructCrossShardTransaction(
	compnetnts *OperationCompnetnts, metadata *ConstructMetadata, sourceShardID uint32,
) (ngyTypes.PoolTransaction, *types.Error) {
	if compnetnts.To == nil {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": "cross shard transfer requires a receiver",
		})
	}
	to, err := getAddress(compnetnts.To)
	if err != nil {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": errors.WithMessage(err, "invalid sender address").Error(),
		})
	}
	if metadata.Transaction.FromShardID == nil || metadata.Transaction.ToShardID == nil {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": "cross shard transfer requires to & from shard IDs",
		})
	}
	if *metadata.Transaction.FromShardID != sourceShardID {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": fmt.Sprintf("cannot send transaction for shard %v", *metadata.Transaction.FromShardID),
		})
	}
	if *metadata.Transaction.FromShardID == *metadata.Transaction.ToShardID {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": "cross-shard transfer cannot be within same shard",
		})
	}
	data := hexutil.Bytes{}
	if metadata.Transaction.Data != nil {
		if data, err = hexutil.Decode(*metadata.Transaction.Data); err != nil {
			return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
				"message": errors.WithMessage(err, "improper data format for transaction data").Error(),
			})
		}
	}
	return ngyTypes.NewCrossShardTransaction(
		metadata.Nonce, &to, *metadata.Transaction.FromShardID, *metadata.Transaction.ToShardID,
		compnetnts.Amount, metadata.GasLimit, metadata.GasPrice, data,
	), nil
}

// constructContractCreationTransaction ..
func constructContractCreationTransaction(
	compnetnts *OperationCompnetnts, metadata *ConstructMetadata, sourceShardID uint32,
) (ngyTypes.PoolTransaction, *types.Error) {
	if metadata.Transaction.Data == nil {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": "contract creation requires data, but nnet found in transaction metadata",
		})
	}
	data, err := hexutil.Decode(*metadata.Transaction.Data)
	if err != nil {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": errors.WithMessage(err, "improper data format for transaction data").Error(),
		})
	}
	return ngyTypes.NewContractCreation(
		metadata.Nonce, sourceShardID, compnetnts.Amount, metadata.GasLimit, metadata.GasPrice, data,
	), nil
}

// constructPlainTransaction ..
func constructPlainTransaction(
	compnetnts *OperationCompnetnts, metadata *ConstructMetadata, sourceShardID uint32,
) (ngyTypes.PoolTransaction, *types.Error) {
	if compnetnts.To == nil {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": "cross shard transfer requires a receiver",
		})
	}
	to, err := getAddress(compnetnts.To)
	if err != nil {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": errors.WithMessage(err, "invalid sender address").Error(),
		})
	}
	data := hexutil.Bytes{}
	if metadata.Transaction.Data != nil {
		if data, err = hexutil.Decode(*metadata.Transaction.Data); err != nil {
			return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
				"message": errors.WithMessage(err, "improper data format for transaction data").Error(),
			})
		}
	}
	return ngyTypes.NewTransaction(
		metadata.Nonce, to, sourceShardID, compnetnts.Amount, metadata.GasLimit, metadata.GasPrice, data,
	), nil
}
