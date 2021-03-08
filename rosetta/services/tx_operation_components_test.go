package services

import (
	"math/big"
	"testing"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/crypto"

	internalCommon "github.com/nordicenergy/nordicenergy-core/internal/common"
	"github.com/nordicenergy/nordicenergy-core/rosetta/common"
)

func TestGetContractCreationOperationCompnetnts(t *testing.T) {
	refAmount := &types.Amount{
		Value:    "-12000",
		Currency: &common.NativeCurrency,
	}
	refKey := internalCommon.MustGeneratePrivateKey()
	refFrom, rosettaError := newAccountIdentifier(crypto.PubkeyToAddress(refKey.PublicKey))
	if rosettaError != nil {
		t.Fatal(rosettaError)
	}

	// test valid operations
	refOperation := &types.Operation{
		Type:    common.ContractCreationOperation,
		Amount:  refAmount,
		Account: refFrom,
	}
	testCompnetnts, rosettaError := getContractCreationOperationCompnetnts(refOperation)
	if rosettaError != nil {
		t.Error(rosettaError)
	}
	if testCompnetnts.Type != refOperation.Type {
		t.Error("expected same operation")
	}
	if testCompnetnts.From == nil || types.Hash(testCompnetnts.From) != types.Hash(refFrom) {
		t.Error("expect same sender")
	}
	if testCompnetnts.Amount.Cmp(big.NewInt(12000)) != 0 {
		t.Error("expected amount to be absolute value of reference amount")
	}

	// test nil amount
	_, rosettaError = getContractCreationOperationCompnetnts(&types.Operation{
		Type:    common.ContractCreationOperation,
		Amount:  nil,
		Account: refFrom,
	})
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test positive amount
	_, rosettaError = getContractCreationOperationCompnetnts(&types.Operation{
		Type: common.ContractCreationOperation,
		Amount: &types.Amount{
			Value:    "12000",
			Currency: &common.NativeCurrency,
		},
		Account: refFrom,
	})
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test different/unsupported currency
	_, rosettaError = getContractCreationOperationCompnetnts(&types.Operation{
		Type: common.ContractCreationOperation,
		Amount: &types.Amount{
			Value: "-12000",
			Currency: &types.Currency{
				Symbol:   "bad",
				Decimals: 9,
			},
		},
		Account: refFrom,
	})
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test nil account
	_, rosettaError = getContractCreationOperationCompnetnts(&types.Operation{
		Type:    common.ContractCreationOperation,
		Amount:  refAmount,
		Account: nil,
	})
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test nil operation
	_, rosettaError = getContractCreationOperationCompnetnts(nil)
	if rosettaError == nil {
		t.Error("expected error")
	}
}

func TestGetCrossShardOperationCompnetnts(t *testing.T) {
	refAmount := &types.Amount{
		Value:    "-12000",
		Currency: &common.NativeCurrency,
	}
	refFromKey := internalCommon.MustGeneratePrivateKey()
	refFrom, rosettaError := newAccountIdentifier(crypto.PubkeyToAddress(refFromKey.PublicKey))
	if rosettaError != nil {
		t.Fatal(rosettaError)
	}
	refToKey := internalCommon.MustGeneratePrivateKey()
	refTo, rosettaError := newAccountIdentifier(crypto.PubkeyToAddress(refToKey.PublicKey))
	if rosettaError != nil {
		t.Fatal(rosettaError)
	}
	refMetadata := common.CrossShardTransactionOperationMetadata{
		From: refFrom,
		To:   refTo,
	}
	refMetadataMap, err := types.MarshalMap(refMetadata)
	if err != nil {
		t.Fatal(err)
	}

	// test valid operations
	refOperation := &types.Operation{
		Type:     common.NativeCrossShardTransferOperation,
		Amount:   refAmount,
		Account:  refFrom,
		Metadata: refMetadataMap,
	}
	testCompnetnts, rosettaError := getCrossShardOperationCompnetnts(refOperation)
	if rosettaError != nil {
		t.Fatal(rosettaError)
	}
	if testCompnetnts.Type != refOperation.Type {
		t.Error("expected same operation")
	}
	if testCompnetnts.From == nil || types.Hash(testCompnetnts.From) != types.Hash(refFrom) {
		t.Error("expect same sender")
	}
	if testCompnetnts.To == nil || types.Hash(testCompnetnts.To) != types.Hash(refTo) {
		t.Error("expected same sender")
	}
	if testCompnetnts.Amount.Cmp(big.NewInt(12000)) != 0 {
		t.Error("expected amount to be absolute value of reference amount")
	}

	// test nil amount
	_, rosettaError = getCrossShardOperationCompnetnts(&types.Operation{
		Type:     common.NativeCrossShardTransferOperation,
		Amount:   nil,
		Account:  refFrom,
		Metadata: refMetadataMap,
	})
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test positive amount
	_, rosettaError = getCrossShardOperationCompnetnts(&types.Operation{
		Type: common.NativeCrossShardTransferOperation,
		Amount: &types.Amount{
			Value:    "12000",
			Currency: &common.NativeCurrency,
		},
		Account:  refFrom,
		Metadata: refMetadataMap,
	})
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test different/unsupported currency
	_, rosettaError = getCrossShardOperationCompnetnts(&types.Operation{
		Type: common.NativeCrossShardTransferOperation,
		Amount: &types.Amount{
			Value: "-12000",
			Currency: &types.Currency{
				Symbol:   "bad",
				Decimals: 9,
			},
		},
		Account:  refFrom,
		Metadata: refMetadataMap,
	})
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test nil account
	_, rosettaError = getCrossShardOperationCompnetnts(&types.Operation{
		Type:     common.NativeCrossShardTransferOperation,
		Amount:   refAmount,
		Account:  nil,
		Metadata: refMetadataMap,
	})
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test no metadata
	_, rosettaError = getCrossShardOperationCompnetnts(&types.Operation{
		Type:    common.NativeCrossShardTransferOperation,
		Amount:  refAmount,
		Account: refFrom,
	})
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test bad metadata
	randomKey := internalCommon.MustGeneratePrivateKey()
	randomID, rosettaError := newAccountIdentifier(crypto.PubkeyToAddress(randomKey.PublicKey))
	if rosettaError != nil {
		t.Fatal(rosettaError)
	}
	badMetadata := common.CrossShardTransactionOperationMetadata{
		From: randomID,
		To:   refTo,
	}
	badMetadataMap, err := types.MarshalMap(badMetadata)
	if err != nil {
		t.Fatal(err)
	}
	_, rosettaError = getCrossShardOperationCompnetnts(&types.Operation{
		Type:     common.NativeCrossShardTransferOperation,
		Amount:   refAmount,
		Account:  refFrom,
		Metadata: badMetadataMap,
	})
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test nil operation
	_, rosettaError = getCrossShardOperationCompnetnts(nil)
	if rosettaError == nil {
		t.Error("expected error")
	}
}

func TestGetTransferOperationCompnetnts(t *testing.T) {
	refFromAmount := &types.Amount{
		Value:    "-12000",
		Currency: &common.NativeCurrency,
	}
	refToAmount := &types.Amount{
		Value:    "12000",
		Currency: &common.NativeCurrency,
	}
	refFromKey := internalCommon.MustGeneratePrivateKey()
	refFrom, rosettaError := newAccountIdentifier(crypto.PubkeyToAddress(refFromKey.PublicKey))
	if rosettaError != nil {
		t.Fatal(rosettaError)
	}
	refToKey := internalCommon.MustGeneratePrivateKey()
	refTo, rosettaError := newAccountIdentifier(crypto.PubkeyToAddress(refToKey.PublicKey))
	if rosettaError != nil {
		t.Fatal(rosettaError)
	}

	// test valid operations
	refOperations := []*types.Operation{
		{
			OperationIdentifier: &types.OperationIdentifier{
				Index: 0,
			},
			Type:    common.NativeTransferOperation,
			Amount:  refFromAmount,
			Account: refFrom,
		},
		{
			OperationIdentifier: &types.OperationIdentifier{
				Index: 1,
			},
			RelatedOperations: []*types.OperationIdentifier{
				{
					Index: 0,
				},
			},
			Type:    common.NativeTransferOperation,
			Amount:  refToAmount,
			Account: refTo,
		},
	}
	testCompnetnts, rosettaError := getTransferOperationCompnetnts(refOperations)
	if rosettaError != nil {
		t.Fatal(rosettaError)
	}
	if testCompnetnts.Type != refOperations[0].Type {
		t.Error("expected same operation")
	}
	if testCompnetnts.From == nil || types.Hash(testCompnetnts.From) != types.Hash(refFrom) {
		t.Error("expect same sender")
	}
	if testCompnetnts.To == nil || types.Hash(testCompnetnts.To) != types.Hash(refTo) {
		t.Error("expected same sender")
	}
	if testCompnetnts.Amount.Cmp(big.NewInt(12000)) != 0 {
		t.Error("expected amount to be absolute value of reference amount")
	}

	// test valid operations flipped
	refOperations[0].Amount = refToAmount
	refOperations[0].Account = refTo
	refOperations[1].Amount = refFromAmount
	refOperations[1].Account = refFrom
	testCompnetnts, rosettaError = getTransferOperationCompnetnts(refOperations)
	if rosettaError != nil {
		t.Fatal(rosettaError)
	}
	if testCompnetnts.Type != refOperations[0].Type {
		t.Error("expected same operation")
	}
	if testCompnetnts.From == nil || types.Hash(testCompnetnts.From) != types.Hash(refFrom) {
		t.Error("expect same sender")
	}
	if testCompnetnts.To == nil || types.Hash(testCompnetnts.To) != types.Hash(refTo) {
		t.Error("expected same sender")
	}
	if testCompnetnts.Amount.Cmp(big.NewInt(12000)) != 0 {
		t.Error("expected amount to be absolute value of reference amount")
	}

	// test no sender
	refOperations[0].Amount = refFromAmount
	refOperations[0].Account = nil
	refOperations[1].Amount = refToAmount
	refOperations[1].Account = refTo
	_, rosettaError = getTransferOperationCompnetnts(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test no receiver
	refOperations[0].Amount = refFromAmount
	refOperations[0].Account = refFrom
	refOperations[1].Amount = refToAmount
	refOperations[1].Account = nil
	_, rosettaError = getTransferOperationCompnetnts(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test invalid operation
	refOperations[0].Type = common.ExpendGasOperation
	refOperations[1].Type = common.NativeTransferOperation
	_, rosettaError = getTransferOperationCompnetnts(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test invalid operation sender
	refOperations[0].Type = common.NativeTransferOperation
	refOperations[1].Type = common.ExpendGasOperation
	_, rosettaError = getTransferOperationCompnetnts(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}
	refOperations[1].Type = common.NativeTransferOperation

	// test nil amount
	refOperations[0].Amount = nil
	refOperations[0].Account = refFrom
	refOperations[1].Amount = refToAmount
	refOperations[1].Account = refTo
	_, rosettaError = getTransferOperationCompnetnts(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test nil amount sender
	refOperations[0].Amount = refFromAmount
	refOperations[0].Account = refFrom
	refOperations[1].Amount = nil
	refOperations[1].Account = refTo
	_, rosettaError = getTransferOperationCompnetnts(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test uneven amount
	refOperations[0].Amount = refFromAmount
	refOperations[0].Account = refFrom
	refOperations[1].Amount = &types.Amount{
		Value:    "0",
		Currency: &common.NativeCurrency,
	}
	refOperations[1].Account = refTo
	_, rosettaError = getTransferOperationCompnetnts(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test uneven amount sender
	refOperations[0].Amount = &types.Amount{
		Value:    "0",
		Currency: &common.NativeCurrency,
	}
	refOperations[0].Account = refFrom
	refOperations[1].Amount = refToAmount
	refOperations[1].Account = refTo
	_, rosettaError = getTransferOperationCompnetnts(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test nil amount
	refOperations[0].Amount = refFromAmount
	refOperations[0].Account = refFrom
	refOperations[1].Amount = nil
	refOperations[1].Account = refTo
	_, rosettaError = getTransferOperationCompnetnts(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test nil amount sender
	refOperations[0].Amount = nil
	refOperations[0].Account = refFrom
	refOperations[1].Amount = refToAmount
	refOperations[1].Account = refTo
	_, rosettaError = getTransferOperationCompnetnts(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test invalid currency
	refOperations[0].Amount = refFromAmount
	refOperations[0].Amount.Currency = &types.Currency{
		Symbol:   "bad",
		Decimals: 9,
	}
	refOperations[0].Account = refFrom
	refOperations[1].Amount = refToAmount
	refOperations[1].Account = refTo
	_, rosettaError = getTransferOperationCompnetnts(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}
	refOperations[0].Amount.Currency = &common.NativeCurrency

	// test invalid currency sender
	refOperations[0].Amount = refFromAmount
	refOperations[0].Account = refFrom
	refOperations[1].Amount = refToAmount
	refOperations[1].Amount.Currency = &types.Currency{
		Symbol:   "bad",
		Decimals: 9,
	}
	refOperations[1].Account = refTo
	_, rosettaError = getTransferOperationCompnetnts(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}
	refOperations[1].Amount.Currency = &common.NativeCurrency

	// test invalid related operation
	refOperations[1].RelatedOperations[0].Index = 2
	_, rosettaError = getTransferOperationCompnetnts(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}
	refOperations[1].RelatedOperations[0].Index = 0

	// test cyclic related operation
	refOperations[0].RelatedOperations = []*types.OperationIdentifier{
		{
			Index: 1,
		},
	}
	_, rosettaError = getTransferOperationCompnetnts(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}

	// Test invalid related operation sender
	refOperations[1].RelatedOperations = nil
	refOperations[0].RelatedOperations[0].Index = 3
	_, rosettaError = getTransferOperationCompnetnts(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}

	// Test nil operations
	_, rosettaError = getTransferOperationCompnetnts(nil)
	if rosettaError == nil {
		t.Error("expected error")
	}
}

func TestGetOperationCompnetnts(t *testing.T) {
	refFromAmount := &types.Amount{
		Value:    "-12000",
		Currency: &common.NativeCurrency,
	}
	refToAmount := &types.Amount{
		Value:    "12000",
		Currency: &common.NativeCurrency,
	}
	refFromKey := internalCommon.MustGeneratePrivateKey()
	refFrom, rosettaError := newAccountIdentifier(crypto.PubkeyToAddress(refFromKey.PublicKey))
	if rosettaError != nil {
		t.Fatal(rosettaError)
	}
	refToKey := internalCommon.MustGeneratePrivateKey()
	refTo, rosettaError := newAccountIdentifier(crypto.PubkeyToAddress(refToKey.PublicKey))
	if rosettaError != nil {
		t.Fatal(rosettaError)
	}

	// test valid transaction operation
	// Detailed test in TestGetTransferOperationCompnetnts
	_, rosettaError = GetOperationCompnetnts([]*types.Operation{
		{
			OperationIdentifier: &types.OperationIdentifier{
				Index: 0,
			},
			Type:    common.NativeTransferOperation,
			Amount:  refFromAmount,
			Account: refFrom,
		},
		{
			OperationIdentifier: &types.OperationIdentifier{
				Index: 1,
			},
			RelatedOperations: []*types.OperationIdentifier{
				{
					Index: 0,
				},
			},
			Type:    common.NativeTransferOperation,
			Amount:  refToAmount,
			Account: refTo,
		},
	})
	if rosettaError != nil {
		t.Error(rosettaError)
	}

	// test valid cross-shard transaction operation
	// Detailed test in TestGetCrossShardOperationCompnetnts
	refMetadata := common.CrossShardTransactionOperationMetadata{
		From: refFrom,
		To:   refTo,
	}
	refMetadataMap, err := types.MarshalMap(refMetadata)
	if err != nil {
		t.Fatal(err)
	}
	_, rosettaError = GetOperationCompnetnts([]*types.Operation{
		{
			Type:     common.NativeCrossShardTransferOperation,
			Amount:   refFromAmount,
			Account:  refFrom,
			Metadata: refMetadataMap,
		},
	})
	if rosettaError != nil {
		t.Error(rosettaError)
	}

	// test valid contract creation operation
	// Detailed test in TestGetContractCreationOperationCompnetnts
	_, rosettaError = GetOperationCompnetnts([]*types.Operation{
		{
			Type:    common.ContractCreationOperation,
			Amount:  refFromAmount,
			Account: refFrom,
		},
	})
	if rosettaError != nil {
		t.Error(rosettaError)
	}

	// test invalid number of operations
	refOperations := []*types.Operation{}
	_, rosettaError = GetOperationCompnetnts(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test invalid number of operations pas max number of operations
	for i := 0; i <= maxNumOfConstructionOps+1; i++ {
		refOperations = append(refOperations, &types.Operation{})
	}
	_, rosettaError = GetOperationCompnetnts(refOperations)
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test invalid operation
	_, rosettaError = GetOperationCompnetnts([]*types.Operation{
		{
			Type: common.ExpendGasOperation,
		},
	})
	if rosettaError == nil {
		t.Error("expected error")
	}

	// test nil operation
	_, rosettaError = GetOperationCompnetnts(nil)
	if rosettaError == nil {
		t.Error("expected error")
	}
}
