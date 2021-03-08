// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package rpc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/nordicenergy/nordicenergy-core/core"
	"github.com/nordicenergy/nordicenergy-core/core/rawdb"
	"github.com/nordicenergy/nordicenergy-core/core/types"
	"github.com/nordicenergy/nordicenergy-core/ngy"
)

const (
	// defaultTraceTimeout is the amount of time a single transaction can execute
	// by default before being forcefully aborted.
	defaultTraceTimeout = 5 * time.Second

	// defaultTraceReExec is the number of blocks the tracer is willing to go back
	// and re-execute to produce missing historical state necessary to run a specific
	// trace.
	defaultTraceReexec = uint64(128)
)

var (
	// ErrNotAvailable to indicate the RPC is not ready for public use
	ErrNotAvailable = errors.New("RPC not available yet")
)

// PublicTracerService provides an API to access nordicenergy's staking services.
// It offers only methods that operate on public data that is freely available to anynet.
type PublicTracerService struct {
	ngy     *ngy.nordicenergy
	version Version
}

// NewPublicTracerAPI creates a new API for the RPC interface
func NewPublicTracerAPI(ngy *ngy.nordicenergy, version Version) rpc.API {
	return rpc.API{
		Namespace: version.Namespace(),
		Version:   APIVersion,
		Service:   &PublicTracerService{ngy, version},
		Public:    true,
	}
}

// TraceChain returns the structured logs created during the execution of EVM
// between two blocks (excluding start) and returns them as a JSON object.
func (s *PublicTracerService) TraceChain(ctx context.Context, start, end rpc.BlockNumber, config *ngy.TraceConfig) (*rpc.Subscription, error) {
	// TODO (JL): Make API available after DoS testing
	return nil, ErrNotAvailable
	if uint64(start) >= uint64(end) {
		return nil, fmt.Errorf("start block can not be equal or greater than the end block")
	}

	currentBlock := s.ngy.BlockChain.CurrentBlock().NumberU64()
	if uint64(start) > currentBlock || uint64(end) > currentBlock {
		return nil, ErrRequestedBlockTooHigh
	}

	from := s.ngy.BlockChain.GetBlockByNumber(uint64(start))
	if from == nil {
		return nil, fmt.Errorf("start block #%d not found", start)
	}
	to := s.ngy.BlockChain.GetBlockByNumber(uint64(end))
	if to == nil {
		return nil, fmt.Errorf("end block #%d not found", end)
	}

	return s.ngy.TraceChain(ctx, from, to, config)
}

// TraceBlockByNumber returns the structured logs created during the execution of
// EVM and returns them as a JSON object.
func (s *PublicTracerService) TraceBlockByNumber(ctx context.Context, number rpc.BlockNumber, config *ngy.TraceConfig) ([]*ngy.TxTraceResult, error) {
	// Fetch the block that we want to trace
	block := s.ngy.BlockChain.GetBlockByNumber(uint64(number))

	return s.ngy.TraceBlock(ctx, block, config)
}

// TraceBlockByHash returns the structured logs created during the execution of
// EVM and returns them as a JSON object.
func (s *PublicTracerService) TraceBlockByHash(ctx context.Context, hash common.Hash, config *ngy.TraceConfig) ([]*ngy.TxTraceResult, error) {
	block := s.ngy.BlockChain.GetBlockByHash(hash)
	if block == nil {
		return nil, fmt.Errorf("block %#x not found", hash)
	}
	return s.ngy.TraceBlock(ctx, block, config)
}

// TraceBlock returns the structured logs created during the execution of EVM
// and returns them as a JSON object.
func (s *PublicTracerService) TraceBlock(ctx context.Context, blob []byte, config *ngy.TraceConfig) ([]*ngy.TxTraceResult, error) {
	block := new(types.Block)
	if err := rlp.Decode(bytes.NewReader(blob), block); err != nil {
		return nil, fmt.Errorf("could not decode block: %v", err)
	}
	return s.ngy.TraceBlock(ctx, block, config)
}

// TraceTransaction returns the structured logs created during the execution of EVM
// and returns them as a JSON object.
func (s *PublicTracerService) TraceTransaction(ctx context.Context, hash common.Hash, config *ngy.TraceConfig) (interface{}, error) {
	// Retrieve the transaction and assemble its EVM context
	tx, blockHash, _, index := rawdb.ReadTransaction(s.ngy.ChainDb(), hash)
	if tx == nil {
		return nil, fmt.Errorf("transaction %#x not found", hash)
	}
	reexec := defaultTraceReexec
	if config != nil && config.Reexec != nil {
		reexec = *config.Reexec
	}
	// Retrieve the block
	block := s.ngy.BlockChain.GetBlockByHash(blockHash)
	if block == nil {
		return nil, fmt.Errorf("block %#x not found", blockHash)
	}
	msg, vmctx, statedb, err := s.ngy.ComputeTxEnv(block, int(index), reexec)
	if err != nil {
		return nil, err
	}
	// Trace the transaction and return
	return s.ngy.TraceTx(ctx, msg, vmctx, statedb, config)
}

// TraceCall lets you trace a given eth_call. It collects the structured logs created during the execution of EVM
// if the given transaction was added on top of the provided block and returns them as a JSON object.
// You can provide -2 as a block number to trace on top of the pending block.
// NOTE: Our version only supports block number as an input
func (s *PublicTracerService) TraceCall(ctx context.Context, args CallArgs, blockNr rpc.BlockNumber, config *ngy.TraceConfig) (interface{}, error) {
	// First try to retrieve the state
	statedb, header, err := s.ngy.StateAndHeaderByNumber(ctx, blockNr)
	if err != nil {
		// Try to retrieve the specified block
		block := s.ngy.BlockChain.GetBlockByNumber(uint64(blockNr))
		if block == nil {
			return nil, fmt.Errorf("block %v not found: %v", blockNr, err)
		}
		// try to recompute the state
		reexec := defaultTraceReexec
		if config != nil && config.Reexec != nil {
			reexec = *config.Reexec
		}
		_, _, statedb, err = s.ngy.ComputeTxEnv(block, 0, reexec)
		if err != nil {
			return nil, err
		}
	}

	// Execute the trace
	msg := args.ToMessage(s.ngy.RPCGasCap)
	vmctx := core.NewEVMContext(msg, header, s.ngy.BlockChain, nil)
	// Trace the transaction and return
	return s.ngy.TraceTx(ctx, msg, vmctx, statedb, config)
}
