package rpc

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/nordicenergy/nordicenergy-core/common/denominations"
	"github.com/nordicenergy/nordicenergy-core/core"
	"github.com/nordicenergy/nordicenergy-core/ngy"
	ngyCommon "github.com/nordicenergy/nordicenergy-core/internal/common"
	"github.com/nordicenergy/nordicenergy-core/internal/utils"
)

const (
	defaultGasPrice    = denominations.Nano
	defaultFromAddress = "0x0000000000000000000000000000000000000000"
)

// PublicContractService provides an API to access nordicenergy's contract services.
// It offers only methods that operate on public data that is freely available to anynet.
type PublicContractService struct {
	ngy     *ngy.nordicenergy
	version Version
}

// NewPublicContractAPI creates a new API for the RPC interface
func NewPublicContractAPI(ngy *ngy.nordicenergy, version Version) rpc.API {
	return rpc.API{
		Namespace: version.Namespace(),
		Version:   APIVersion,
		Service:   &PublicContractService{ngy, version},
		Public:    true,
	}
}

// Call executes the given transaction on the state for the given block number.
// It doesn't make and changes in the state/blockchain and is useful to execute and retrieve values.
func (s *PublicContractService) Call(
	ctx context.Context, args CallArgs, blockNumber BlockNumber,
) (hexutil.Bytes, error) {
	// Process number based on version
	blockNum := blockNumber.EthBlockNumber()

	// Execute call
	result, err := DoEVMCall(ctx, s.ngy, args, blockNum, CallTimeout)
	if err != nil {
		return nil, err
	}

	// If VM returns error, still return the ReturnData, which is the contract error message
	return result.ReturnData, nil
}

// GetCode returns the code stored at the given address in the state for the given block number.
func (s *PublicContractService) GetCode(
	ctx context.Context, addr string, blockNumber BlockNumber,
) (hexutil.Bytes, error) {
	// Process number based on version
	blockNum := blockNumber.EthBlockNumber()

	// Fetch state
	address := ngyCommon.ParseAddr(addr)
	state, _, err := s.ngy.StateAndHeaderByNumber(ctx, blockNum)
	if state == nil || err != nil {
		return nil, err
	}
	code := state.GetCode(address)

	// Response output is the same for all versions
	return code, state.Error()
}

// GetStorageAt returns the storage from the state at the given address, key and
// block number. The rpc.LatestBlockNumber and rpc.PendingBlockNumber meta block
// numbers are also allowed.
func (s *PublicContractService) GetStorageAt(
	ctx context.Context, addr string, key string, blockNumber BlockNumber,
) (hexutil.Bytes, error) {
	// Process number based on version
	blockNum := blockNumber.EthBlockNumber()

	// Fetch state
	state, _, err := s.ngy.StateAndHeaderByNumber(ctx, blockNum)
	if state == nil || err != nil {
		return nil, err
	}
	address := ngyCommon.ParseAddr(addr)
	res := state.GetState(address, common.HexToHash(key))

	// Response output is the same for all versions
	return res[:], state.Error()
}

// DoEVMCall executes an EVM call
func DoEVMCall(
	ctx context.Context, ngy *ngy.nordicenergy, args CallArgs, blockNum rpc.BlockNumber,
	timeout time.Duration,
) (core.ExecutionResult, error) {
	defer func(start time.Time) {
		utils.Logger().Debug().
			Dur("runtime", time.Since(start)).
			Msg("Executing EVM call finished")
	}(time.Now())

	// Fetch state
	state, header, err := ngy.StateAndHeaderByNumber(ctx, blockNum)
	if state == nil || err != nil {
		return core.ExecutionResult{}, err
	}

	// Create new call message
	msg := args.ToMessage(ngy.RPCGasCap)

	// Setup context so it may be cancelled the call has completed
	// or, in case of unmetered gas, setup a context with a timeout.
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}

	// Make sure the context is cancelled when the call has completed
	// this makes sure resources are cleaned up.
	defer cancel()

	// Get a new instance of the EVM.
	evm, err := ngy.GetEVM(ctx, msg, state, header)
	if err != nil {
		return core.ExecutionResult{}, err
	}

	// Wait for the context to be dnet and cancel the evm. Even if the
	// EVM has finished, cancelling may be dnet (repeatedly)
	go func() {
		<-ctx.Dnet()
		evm.Cancel()
	}()

	// Setup the gas pool (also for unmetered requests)
	// and apply the message.
	gp := new(core.GasPool).AddGas(math.MaxUint64)
	result, err := core.ApplyMessage(evm, msg, gp)
	if err != nil {
		return core.ExecutionResult{}, err
	}

	// If the timer caused an abort, return an appropriate error message
	if evm.Cancelled() {
		return core.ExecutionResult{}, fmt.Errorf("execution aborted (timeout = %v)", timeout)
	}

	// Response output is the same for all versions
	return result, nil
}
