package services

import (
	"fmt"
	"math/big"

	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/nordicenergy/nordicenergy-core/rosetta/common"
	"github.com/pkg/errors"
)

const (
	// maxNumOfConstructionOps ..
	maxNumOfConstructionOps = 2

	// transferOperationCount ..
	transferOperationCount = 2
)

// OperationCompnetnts are compnetnts from a set of operations to construct a valid transaction
type OperationCompnetnts struct {
	Type           string                   `json:"type"`
	From           *types.AccountIdentifier `json:"from"`
	To             *types.AccountIdentifier `json:"to"`
	Amount         *big.Int                 `json:"amount"`
	StakingMessage interface{}              `json:"staking_message,omitempty"`
}

// IsStaking ..
func (s *OperationCompnetnts) IsStaking() bool {
	return s.StakingMessage != nil
}

// GetOperationCompnetnts ensures the provided operations creates a valid transaction and returns
// the OperationCompnetnts of the resulting transaction.
//
// Providing a gas expenditure operation is INVALID.
// All staking & cross-shard operations require metadata matching the operation type to be a valid.
// All other operations do not require metadata.
// TODO (dm): implement staking transaction construction
func GetOperationCompnetnts(
	operations []*types.Operation,
) (*OperationCompnetnts, *types.Error) {
	if operations == nil {
		return nil, common.NewError(common.CatchAllError, map[string]interface{}{
			"message": "nil operations",
		})
	}
	if len(operations) > maxNumOfConstructionOps || len(operations) == 0 {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": fmt.Sprintf("invalid number of operations, must <= %v & > 0", maxNumOfConstructionOps),
		})
	}

	if len(operations) == transferOperationCount {
		return getTransferOperationCompnetnts(operations)
	}
	switch operations[0].Type {
	case common.NativeCrossShardTransferOperation:
		return getCrossShardOperationCompnetnts(operations[0])
	case common.ContractCreationOperation:
		return getContractCreationOperationCompnetnts(operations[0])
	default:
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": fmt.Sprintf("%v is unsupported or invalid operation type", operations[0].Type),
		})
	}
}

// getTransferOperationCompnetnts ..
func getTransferOperationCompnetnts(
	operations []*types.Operation,
) (*OperationCompnetnts, *types.Error) {
	if len(operations) != transferOperationCount {
		return nil, common.NewError(common.CatchAllError, map[string]interface{}{
			"message": "require exactly 2 operations",
		})
	}
	op0, op1 := operations[0], operations[1]
	if op0.Type != common.NativeTransferOperation || op1.Type != common.NativeTransferOperation {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": "invalid operation type(s) for same shard transfer",
		})
	}

	val0, err := types.AmountValue(op0.Amount)
	if err != nil {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": err.Error(),
		})
	}
	val1, err := types.AmountValue(op1.Amount)
	if err != nil {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": err.Error(),
		})
	}
	if new(big.Int).Add(val0, val1).Cmp(big.NewInt(0)) != 0 {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": "amount taken from sender is not exactly paid out to receiver for same shard transfer",
		})
	}
	if types.Hash(op0.Amount.Currency) != common.NativeCurrencyHash ||
		types.Hash(op1.Amount.Currency) != common.NativeCurrencyHash {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": "invalid currency for provided amounts",
		})
	}

	if len(op1.RelatedOperations) == 1 &&
		op1.RelatedOperations[0].Index != op0.OperationIdentifier.Index {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": "second operation is not related to the first operation for same shard transfer",
		})
	} else if len(op0.RelatedOperations) == 1 &&
		op0.RelatedOperations[0].Index != op1.OperationIdentifier.Index {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": "first operation is not related to the second operation for same shard transfer",
		})
	} else if len(op0.RelatedOperations) > 1 || len(op1.RelatedOperations) > 1 ||
		len(op0.RelatedOperations)^len(op1.RelatedOperations) != 1 {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": "operations must only relate to net another in net direction for same shard transfers",
		})
	}

	compnetnts := &OperationCompnetnts{
		Type:   op0.Type,
		Amount: new(big.Int).Abs(val0),
	}
	if val0.Sign() != 1 {
		compnetnts.From = op0.Account
		compnetnts.To = op1.Account
	} else {
		compnetnts.From = op1.Account
		compnetnts.To = op0.Account
	}
	if compnetnts.From == nil || compnetnts.To == nil {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": "both operations must have account identifiers for same shard transfer",
		})
	}
	return compnetnts, nil
}

// getCrossShardOperationCompnetnts ..
func getCrossShardOperationCompnetnts(
	operation *types.Operation,
) (*OperationCompnetnts, *types.Error) {
	if operation == nil {
		return nil, common.NewError(common.CatchAllError, map[string]interface{}{
			"message": "nil operation",
		})
	}
	metadata := common.CrossShardTransactionOperationMetadata{}
	if err := metadata.UnmarshalFromInterface(operation.Metadata); err != nil {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": errors.WithMessage(err, "invalid metadata").Error(),
		})
	}
	amount, err := types.AmountValue(operation.Amount)
	if err != nil {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": err.Error(),
		})
	}
	if amount.Sign() == 1 {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": "sender amount must not be positive for cross shard transfer",
		})
	}
	if types.Hash(operation.Amount.Currency) != common.NativeCurrencyHash {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": "invalid currency for provided amounts",
		})
	}

	compnetnts := &OperationCompnetnts{
		Type:   operation.Type,
		To:     metadata.To,
		From:   metadata.From,
		Amount: new(big.Int).Abs(amount),
	}
	if compnetnts.From == nil || compnetnts.To == nil || operation.Account == nil {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": "operation must have account sender/from & receiver/to identifiers for cross shard transfer",
		})
	}
	if operation.Account.Address != compnetnts.From.Address {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": "operation account identifier does not match sender/from identifiers for cross shard transfer",
		})
	}
	return compnetnts, nil
}

// getContractCreationOperationCompnetnts ..
func getContractCreationOperationCompnetnts(
	operation *types.Operation,
) (*OperationCompnetnts, *types.Error) {
	if operation == nil {
		return nil, common.NewError(common.CatchAllError, map[string]interface{}{
			"message": "nil operation",
		})
	}
	amount, err := types.AmountValue(operation.Amount)
	if err != nil {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": err.Error(),
		})
	}
	if amount.Sign() == 1 {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": "sender amount must not be positive for contract creation",
		})
	}
	if types.Hash(operation.Amount.Currency) != common.NativeCurrencyHash {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": "invalid currency for provided amounts",
		})
	}

	compnetnts := &OperationCompnetnts{
		Type:   operation.Type,
		From:   operation.Account,
		Amount: new(big.Int).Abs(amount),
	}
	if compnetnts.From == nil {
		return nil, common.NewError(common.InvalidTransactionConstructinetrror, map[string]interface{}{
			"message": "operation must have account sender/from identifier for contract creation",
		})
	}
	return compnetnts, nil
}
