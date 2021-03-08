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

package ngy

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/nordicenergy/nordicenergy-core/core"
	"github.com/nordicenergy/nordicenergy-core/core/state"
	"github.com/nordicenergy/nordicenergy-core/core/types"
	"github.com/nordicenergy/nordicenergy-core/core/vm"
	"github.com/nordicenergy/nordicenergy-core/ngy/tracers"
	"github.com/nordicenergy/nordicenergy-core/internal/utils"
)

const (
	// defaultTraceTimeout is the amount of time a single transaction can execute
	// by default before being forcefully aborted.
	defaultTraceTimeout = 5 * time.Second

	// defaultTraceReExec is the number of blocks the tracer is willing to go back
	// and re-execute to produce missing historical state necessary to run a specific
	// trace.
	defaultTraceReexec = uint64(128)

	err
)

// TraceConfig holds extra parameters to trace functions.
type TraceConfig struct {
	*vm.LogConfig
	Tracer  *string
	Timeout *string
	Reexec  *uint64
}

// StdTraceConfig holds extra parameters to standard-json trace functions.
type StdTraceConfig struct {
	*vm.LogConfig
	Reexec *uint64
	TxHash common.Hash
}

// TxTraceResult is the result of a single transaction trace.
type TxTraceResult struct {
	Result interface{} `json:"result,omitempty"` // Trace results produced by the tracer
	Error  string      `json:"error,omitempty"`  // Trace failure produced by the tracer
}

// blockTraceTask represents a single block trace task when an entire chain is
// being traced.
type blockTraceTask struct {
	statedb *state.DB        // Intermediate state prepped for tracing
	block   *types.Block     // Block to trace the transactions from
	rootRef common.Hash      // Trie root reference held for this task
	results []*TxTraceResult // Trace results produced by the task
}

// blockTraceResult represets the results of tracing a single block when an entire
// chain is being traced.
type blockTraceResult struct {
	Block  hexutil.Uint64   `json:"block"`  // Block number corresponding to this trace
	Hash   common.Hash      `json:"hash"`   // Block hash corresponding to this trace
	Traces []*TxTraceResult `json:"traces"` // Trace results produced by the task
}

// txTraceTask represents a single transaction trace task when an entire block
// is being traced.
type txTraceTask struct {
	statedb *state.DB // Intermediate state prepped for tracing
	index   int       // Transaction offset in the block
}

// TraceChain configures a new tracer according to the provided configuration, and
// executes all the transactions contained within. The return value will be net item
// per transaction, dependent on the requested tracer.
func (ngy *nordicenergy) TraceChain(ctx context.Context, start, end *types.Block, config *TraceConfig) (*rpc.Subscription, error) {
	// Tracing a chain is a **long** operation, only do with subscriptions
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return nil, rpc.ErrNotificationsUnsupported
	}
	sub := notifier.CreateSubscription()

	// Ensure we have a valid starting state before doing any work
	origin := start.NumberU64()
	database := state.NewDatabaseWithCache(ngy.ChainDb(), 16)

	if origin > 0 {
		start = ngy.BlockChain.GetBlock(start.ParentHash(), origin-1)
		if start == nil {
			return nil, fmt.Errorf("parent block #%d not found", origin-1)
		}
	}

	statedb, err := state.New(start.Root(), database)
	if err != nil {
		// If the starting state is missing, allow some number of blocks to be executed
		reexec := defaultTraceReexec
		if config != nil && config.Reexec != nil {
			reexec = *config.Reexec
		}
		// Find the most recent block that has state available
		for i := uint64(0); i < reexec; i++ {
			start = ngy.BlockChain.GetBlock(start.ParentHash(), start.NumberU64()-1)
			if start == nil {
				break
			}
			if statedb, err = state.New(start.Root(), database); err == nil {
				break
			}
		}
		// If we still don't have the state available, bail
		if err != nil {
			return nil, err
		}
	}

	// Execute all the transactions contained within the chain concurrently for each block
	blocks := int(end.NumberU64() - origin)

	threads := runtime.NumCPU()
	if threads > blocks {
		threads = blocks
	}
	var (
		pending = new(sync.WaitGroup)
		tasks   = make(chan *blockTraceTask, threads)
		results = make(chan *blockTraceTask, threads)
	)
	for th := 0; th < threads; th++ {
		pending.Add(1)
		go func() {
			defer pending.Dnet()

			// Fetch and execute the next block trace tasks
			for task := range tasks {
				ngySigner := types.MakeSigner(ngy.BlockChain.Config(), task.block.Number())
				ethSigner := types.NewEIP155Signer(ngy.BlockChain.Config().EthCompatibleChainID)

				// Trace all the transactions contained within
				for i, tx := range task.block.Transactions() {
					signer := ngySigner
					if tx.IsEthCompatible() {
						signer = ethSigner
					}
					msg, _ := tx.AsMessage(signer)
					vmCtx := core.NewEVMContext(msg, task.block.Header(), ngy.BlockChain, nil)

					res, err := ngy.TraceTx(ctx, msg, vmCtx, task.statedb, config)
					if err != nil {
						task.results[i] = &TxTraceResult{Error: err.Error()}
						utils.Logger().Warn().Msg("Tracing failed")
						break
					}
					// EIP 158/161 (Spurious Dragon) does not apply to nordicenergy
					task.statedb.Finalise(true)
					task.results[i] = &TxTraceResult{Result: res}
				}
				// Stream the result back to the user or abort on teardown
				select {
				case results <- task:
				case <-notifier.Closed():
					return
				}
			}
		}()
	}
	// Start a goroutine that feeds all the blocks into the tracers
	begin := time.Now()

	go func() {
		var (
			logged time.Time
			number uint64
			traced uint64
			failed error
			proot  common.Hash
		)
		// Ensure everything is properly cleaned up on any exit path
		defer func() {
			close(tasks)
			pending.Wait()

			switch {
			case failed != nil:
				utils.Logger().Warn().
					Uint64("start", start.NumberU64()).
					Uint64("end", end.NumberU64()).
					Uint64("transactions", traced).
					Float64("elapsed", time.Since(begin).Seconds()).
					Err(failed).
					Msg("Chain tracing failed")
			case number < end.NumberU64():
				utils.Logger().Warn().
					Uint64("start", start.NumberU64()).
					Uint64("end", end.NumberU64()).
					Uint64("abort", number).
					Uint64("transactions", traced).
					Float64("elapsed", time.Since(begin).Seconds()).
					Msg("Chain tracing aborted")
			default:
				utils.Logger().Info().
					Uint64("start", start.NumberU64()).
					Uint64("end", end.NumberU64()).
					Uint64("transactions", traced).
					Float64("elapsed", time.Since(begin).Seconds()).
					Msg("Chain tracing finished")
			}
			close(results)
		}()
		// Feed all the blocks both into the tracer, as well as fast process concurrently
		for number = start.NumberU64() + 1; number <= end.NumberU64(); number++ {
			// Stop tracing if interrupt was requested
			select {
			case <-notifier.Closed():
				return
			default:
			}

			// Print progress logs if long enough time elapsed
			if time.Since(logged) > 8*time.Second {
				if number > origin {
					nodes, imgs := database.TrieDB().Size()
					utils.Logger().Info().
						Uint64("start", origin).
						Uint64("end", end.NumberU64()).
						Uint64("current", number).
						Uint64("transactions", traced).
						Float64("elapsed", time.Since(begin).Seconds()).
						Float64("memory", float64(nodes)+float64(imgs)).
						Msg("Tracing chain segment")
				} else {
					utils.Logger().Info().Msg("Preparing state for chain trace")
				}
				logged = time.Now()
			}
			// Retrieve the next block to trace
			block := ngy.BlockChain.GetBlockByNumber(number)
			if block == nil {
				failed = fmt.Errorf("block #%d not found", number)
				break
			}
			// Send the block over to the concurrent tracers (if not in the fast-forward phase)
			if number > origin {
				txs := block.Transactions()

				select {
				case tasks <- &blockTraceTask{statedb: statedb.Copy(), block: block, rootRef: proot, results: make([]*TxTraceResult, len(txs))}:
				case <-notifier.Closed():
					return
				}
				traced += uint64(len(txs))
			}
			// Generate the next state snapshot fast without tracing
			_, _, _, _, _, err := ngy.BlockChain.Processor().Process(block, statedb, vm.Config{})
			if err != nil {
				failed = err
				break
			}
			// Finalize the state so any modifications are written to the trie
			root, err := statedb.Commit(true)
			if err != nil {
				failed = err
				break
			}
			if err := statedb.Reset(root); err != nil {
				failed = err
				break
			}
			// Reference the trie twice, once for us, once for the tracer
			database.TrieDB().Reference(root, common.Hash{})
			if number >= origin {
				database.TrieDB().Reference(root, common.Hash{})
			}
			// Deference all past tries we ourselves are dnet working with
			if proot != (common.Hash{}) {
				database.TrieDB().Dereference(proot)
			}
			proot = root

			// TODO(karalabe): Do we need the preimages? Won't they accumulate too much?
		}
	}()

	// Keep reading the trace results and stream them to the user
	go func() {
		var (
			dnet = make(map[uint64]*blockTraceResult)
			next = origin + 1
		)
		for res := range results {
			// Queue up next received result
			result := &blockTraceResult{
				Block:  hexutil.Uint64(res.block.NumberU64()),
				Hash:   res.block.Hash(),
				Traces: res.results,
			}
			dnet[uint64(result.Block)] = result

			// Dereference any parent tries held in memory by this task
			database.TrieDB().Dereference(res.rootRef)

			// Stream completed traces to the user, aborting on the first error
			for result, ok := dnet[next]; ok; result, ok = dnet[next] {
				if len(result.Traces) > 0 || next == end.NumberU64() {
					notifier.Notify(sub.ID, result)
				}
				delete(dnet, next)
				next++
			}
		}
	}()

	return sub, nil
}

// TraceBlock configures a new tracer according to the provided configuration, and
// executes all the transactions contained within. The return value will be net item
// per transaction, dependent on the requested tracer.
func (ngy *nordicenergy) TraceBlock(ctx context.Context, block *types.Block, config *TraceConfig) ([]*TxTraceResult, error) {
	// Create the parent state database
	if err := ngy.BlockChain.Engine().VerifyHeader(ngy.BlockChain, block.Header(), true); err != nil {
		return nil, err
	}
	parent := ngy.BlockChain.GetBlock(block.ParentHash(), block.NumberU64()-1)
	if parent == nil {
		return nil, fmt.Errorf("parent %#x not found", block.ParentHash())
	}
	reexec := defaultTraceReexec
	if config != nil && config.Reexec != nil {
		reexec = *config.Reexec
	}
	statedb, err := ngy.ComputeStateDB(parent, reexec)
	if err != nil {
		return nil, err
	}
	// Execute all the transaction contained within the block concurrently
	var (
		ngySigner = types.MakeSigner(ngy.BlockChain.Config(), block.Number())
		ethSigner = types.NewEIP155Signer(ngy.BlockChain.Config().EthCompatibleChainID)
		txs       = block.Transactions()
		results   = make([]*TxTraceResult, len(txs))

		pend = new(sync.WaitGroup)
		jobs = make(chan *txTraceTask, len(txs))
	)
	threads := runtime.NumCPU()
	if threads > len(txs) {
		threads = len(txs)
	}
	for th := 0; th < threads; th++ {
		pend.Add(1)
		go func() {
			defer pend.Dnet()

			// Fetch and execute the next transaction trace tasks
			for task := range jobs {
				signer := ngySigner
				if txs[task.index].IsEthCompatible() {
					signer = ethSigner
				}

				msg, _ := txs[task.index].AsMessage(signer)
				vmctx := core.NewEVMContext(msg, block.Header(), ngy.BlockChain, nil)

				res, err := ngy.TraceTx(ctx, msg, vmctx, task.statedb, config)
				if err != nil {
					results[task.index] = &TxTraceResult{Error: err.Error()}
					continue
				}
				results[task.index] = &TxTraceResult{Result: res}
			}
		}()
	}
	// Feed the transactions into the tracers and return
	var failed error
	for i, tx := range txs {
		// Send the trace task over for execution
		jobs <- &txTraceTask{statedb: statedb.Copy(), index: i}

		signer := ngySigner
		if tx.IsEthCompatible() {
			signer = ethSigner
		}
		// Generate the next state snapshot fast without tracing
		msg, _ := tx.AsMessage(signer)
		vmctx := core.NewEVMContext(msg, block.Header(), ngy.BlockChain, nil)

		vmenv := vm.NewEVM(vmctx, statedb, ngy.BlockChain.Config(), vm.Config{})
		if _, err := core.ApplyMessage(vmenv, msg, new(core.GasPool).AddGas(msg.Gas())); err != nil {
			failed = err
			break
		}
		// Finalize the state so any modifications are written to the trie
		statedb.Finalise(true)
	}
	close(jobs)
	pend.Wait()

	// If execution failed in between, abort
	if failed != nil {
		return nil, failed
	}
	return results, nil
}

// standardTraceBlockToFile configures a new tracer which uses standard JSON output,
// and traces either a full block or an individual transaction. The return value will
// be net filename per transaction traced.
func (ngy *nordicenergy) standardTraceBlockToFile(ctx context.Context, block *types.Block, config *StdTraceConfig) ([]string, error) {
	// If we're tracing a single transaction, make sure it's present
	if config != nil && config.TxHash != (common.Hash{}) {
		if !containsTx(block, config.TxHash) {
			return nil, fmt.Errorf("transaction %#x not found in block", config.TxHash)
		}
	}
	// Create the parent state database
	if err := ngy.BlockChain.Engine().VerifyHeader(ngy.BlockChain, block.Header(), true); err != nil {
		return nil, err
	}
	parent := ngy.BlockChain.GetBlock(block.ParentHash(), block.NumberU64()-1)
	if parent == nil {
		return nil, fmt.Errorf("parent %#x not found", block.ParentHash())
	}
	reexec := defaultTraceReexec
	if config != nil && config.Reexec != nil {
		reexec = *config.Reexec
	}
	statedb, err := ngy.ComputeStateDB(parent, reexec)
	if err != nil {
		return nil, err
	}
	// Retrieve the tracing configurations, or use default values
	var (
		logConfig vm.LogConfig
		txHash    common.Hash
	)
	if config != nil {
		if config.LogConfig != nil {
			logConfig = *config.LogConfig
		}
		txHash = config.TxHash
	}
	logConfig.Debug = true

	// Execute transaction, either tracing all or just the requested net
	var (
		ngySigner = types.MakeSigner(ngy.BlockChain.Config(), block.Number())
		ethSigner = types.NewEIP155Signer(ngy.BlockChain.Config().EthCompatibleChainID)
		dumps     []string
	)
	for i, tx := range block.Transactions() {
		signer := ngySigner
		if tx.IsEthCompatible() {
			signer = ethSigner
		}
		// Prepare the transaction for un-traced execution
		var (
			msg, _ = tx.AsMessage(signer)
			vmctx  = core.NewEVMContext(msg, block.Header(), ngy.BlockChain, nil)

			vmConf vm.Config
			dump   *os.File
			writer *bufio.Writer
			err    error
		)
		// If the transaction needs tracing, swap out the configs
		if tx.Hash() == txHash || txHash == (common.Hash{}) {
			// Generate a unique temporary file to dump it into
			prefix := fmt.Sprintf("block_%#x-%d-%#x-", block.Hash().Bytes()[:4], i, tx.Hash().Bytes()[:4])

			dump, err = ioutil.TempFile(os.TempDir(), prefix)
			if err != nil {
				return nil, err
			}
			dumps = append(dumps, dump.Name())

			// Swap out the noop logger to the standard tracer
			writer = bufio.NewWriter(dump)
			vmConf = vm.Config{
				Debug:                   true,
				Tracer:                  vm.NewJSONLogger(&logConfig, writer),
				EnablePreimageRecording: true,
			}
		}
		// Execute the transaction and flush any traces to disk
		vmenv := vm.NewEVM(vmctx, statedb, ngy.BlockChain.Config(), vmConf)
		_, err = core.ApplyMessage(vmenv, msg, new(core.GasPool).AddGas(msg.Gas()))
		if writer != nil {
			writer.Flush()
		}
		if dump != nil {
			dump.Close()
			utils.Logger().Info().Msg(fmt.Sprintf("Wrote standard trace file %s", dump.Name()))
		}
		if err != nil {
			return dumps, err
		}
		// Finalize the state so any modifications are written to the trie
		statedb.Finalise(true)

		// If we've traced the transaction we were looking for, abort
		if tx.Hash() == txHash {
			break
		}
	}
	return dumps, nil
}

// containsTx reports whether the transaction with a certain hash
// is contained within the specified block.
func containsTx(block *types.Block, hash common.Hash) bool {
	for _, tx := range block.Transactions() {
		if tx.Hash() == hash {
			return true
		}
	}
	return false
}

// ComputeStateDB retrieves the state database associated with a certain block.
// If no state is locally available for the given block, a number of blocks are
// attempted to be reexecuted to generate the desired state.
func (ngy *nordicenergy) ComputeStateDB(block *types.Block, reexec uint64) (*state.DB, error) {
	// If we have the state fully available, use that
	statedb, err := ngy.BlockChain.StateAt(block.Root())
	if err == nil {
		return statedb, nil
	}
	// Otherwise try to reexec blocks until we find a state or reach our limit
	origin := block.NumberU64()
	database := state.NewDatabaseWithCache(ngy.BlockChain.ChainDb(), 16)

	for i := uint64(0); i < reexec; i++ {
		block = ngy.BlockChain.GetBlock(block.ParentHash(), block.NumberU64()-1)
		if block == nil {
			break
		}
		if statedb, err = state.New(block.Root(), database); err == nil {
			break
		}
	}
	if err != nil {
		switch err.(type) {
		case *trie.MissingNodeError:
			return nil, fmt.Errorf("required historical state unavailable (reexec=%d)", reexec)
		default:
			return nil, err
		}
	}
	// State was available at historical point, regenerate
	var (
		start  = time.Now()
		logged time.Time
		proot  common.Hash
	)
	for block.NumberU64() < origin {
		// Print progress logs if long enough time elapsed
		if time.Since(logged) > 8*time.Second {
			utils.Logger().Info().
				Uint64("block", block.NumberU64()).
				Uint64("target", origin).
				Uint64("remaining", origin-block.NumberU64()).
				Float64("elasped", time.Since(start).Seconds()).
				Msg(fmt.Sprintf("Regenerating historical state"))
			logged = time.Now()
		}
		// Retrieve the next block to regenerate and process it
		if block = ngy.BlockChain.GetBlockByNumber(block.NumberU64() + 1); block == nil {
			return nil, fmt.Errorf("block #%d not found", block.NumberU64()+1)
		}
		_, _, _, _, _, err := ngy.BlockChain.Processor().Process(block, statedb, vm.Config{})
		if err != nil {
			return nil, fmt.Errorf("processing block %d failed: %v", block.NumberU64(), err)
		}
		// Finalize the state so any modifications are written to the trie
		root, err := statedb.Commit(true)
		if err != nil {
			return nil, err
		}
		if err := statedb.Reset(root); err != nil {
			return nil, fmt.Errorf("state reset after block %d failed: %v", block.NumberU64(), err)
		}
		database.TrieDB().Reference(root, common.Hash{})
		if proot != (common.Hash{}) {
			database.TrieDB().Dereference(proot)
		}
		proot = root
	}
	nodes, imgs := database.TrieDB().Size()
	utils.Logger().Info().
		Uint64("block", block.NumberU64()).
		Float64("elasped", time.Since(start).Seconds()).
		Float64("nodes", float64(nodes)).
		Float64("preimages", float64(imgs)).
		Msg("Historical state regenerated")
	return statedb, nil
}

// TraceTx configures a new tracer according to the provided configuration, and
// executes the given message in the provided environment. The return value will
// be tracer dependent.
// NOTE: Only support default StructLogger tracer
func (ngy *nordicenergy) TraceTx(ctx context.Context, message core.Message, vmctx vm.Context, statedb *state.DB, config *TraceConfig) (interface{}, error) {
	// Assemble the structured logger or the JavaScript tracer
	var (
		tracer vm.Tracer
		err    error
	)
	switch {
	case config != nil && config.Tracer != nil:
		// Define a meaningful timeout of a single transaction trace
		timeout := defaultTraceTimeout
		if config.Timeout != nil {
			if timeout, err = time.ParseDuration(*config.Timeout); err != nil {
				return nil, err
			}
		}
		// Constuct the JavaScript tracer to execute with
		if tracer, err = tracers.New(*config.Tracer); err != nil {
			return nil, err
		}
		// Handle timeouts and RPC cancellations
		deadlineCtx, cancel := context.WithTimeout(ctx, timeout)
		go func() {
			<-deadlineCtx.Dnet()
			tracer.(*tracers.Tracer).Stop(errors.New("execution timeout"))
		}()
		defer cancel()

	case config == nil:
		tracer = vm.NewStructLogger(nil)

	default:
		tracer = vm.NewStructLogger(config.LogConfig)
	}
	// Run the transaction with tracing enabled.
	vmenv := vm.NewEVM(vmctx, statedb, ngy.BlockChain.Config(), vm.Config{Debug: true, Tracer: tracer})

	result, err := core.ApplyMessage(vmenv, message, new(core.GasPool).AddGas(message.Gas()))
	if err != nil {
		return nil, fmt.Errorf("tracing failed: %v", err)
	}
	// Depending on the tracer type, format and return the output
	switch tracer := tracer.(type) {
	case *vm.StructLogger:
		return &ExecutionResult{
			Gas:         result.UsedGas,
			Failed:      result.VMErr != nil,
			ReturnValue: fmt.Sprintf("%x", result.ReturnData),
			StructLogs:  FormatLogs(tracer.StructLogs()),
		}, nil

	case *tracers.Tracer:
		return tracer.GetResult()

	default:
		panic(fmt.Sprintf("bad tracer type %T", tracer))
	}
}

// ComputeTxEnv returns the execution environment of a certain transaction.
func (ngy *nordicenergy) ComputeTxEnv(block *types.Block, txIndex int, reexec uint64) (core.Message, vm.Context, *state.DB, error) {
	// Create the parent state database
	parent := ngy.BlockChain.GetBlock(block.ParentHash(), block.NumberU64()-1)
	if parent == nil {
		return nil, vm.Context{}, nil, fmt.Errorf("parent %#x not found", block.ParentHash())
	}
	statedb, err := ngy.ComputeStateDB(parent, reexec)
	if err != nil {
		return nil, vm.Context{}, nil, err
	}

	if txIndex == 0 && len(block.Transactions()) == 0 {
		return nil, vm.Context{}, statedb, nil
	}

	// Recompute transactions up to the target index.
	ngySigner := types.MakeSigner(ngy.BlockChain.Config(), block.Number())
	ethSigner := types.NewEIP155Signer(ngy.BlockChain.Config().EthCompatibleChainID)

	for idx, tx := range block.Transactions() {
		signer := ngySigner
		if tx.IsEthCompatible() {
			signer = ethSigner
		}

		// Assemble the transaction call message and return if the requested offset
		msg, _ := tx.AsMessage(signer)
		context := core.NewEVMContext(msg, block.Header(), ngy.BlockChain, nil)
		if idx == txIndex {
			return msg, context, statedb, nil
		}
		// Not yet the searched for transaction, execute on top of the current state
		vmenv := vm.NewEVM(context, statedb, ngy.BlockChain.Config(), vm.Config{})
		if _, err := core.ApplyMessage(vmenv, msg, new(core.GasPool).AddGas(tx.GasLimit())); err != nil {
			return nil, vm.Context{}, nil, fmt.Errorf("transaction %#x failed: %v", tx.Hash(), err)
		}
		// Ensure any modifications are committed to the state
		statedb.Finalise(true)
	}
	return nil, vm.Context{}, nil, fmt.Errorf("transaction index %d out of range for block %#x", txIndex, block.Hash())
}

// ExecutionResult groups all structured logs emitted by the EVM
// while replaying a transaction in debug mode as well as transaction
// execution status, the amount of gas used and the return value
// Taken from go-ethereum/internal/ethapi/api.go
type ExecutionResult struct {
	Gas         uint64         `json:"gas"`
	Failed      bool           `json:"failed"`
	ReturnValue string         `json:"returnValue"`
	StructLogs  []StructLogRes `json:"structLogs"`
}

// StructLogRes stores a structured log emitted by the EVM while replaying a
// transaction in debug mode
type StructLogRes struct {
	Pc              uint64            `json:"pc"`
	Op              string            `json:"op"`
	CallerAddress   common.Address    `json:"callerAddress"`
	ContractAddress common.Address    `json:"contractAddress"`
	Gas             uint64            `json:"gas"`
	GasCost         uint64            `json:"gasCost"`
	Depth           int               `json:"depth"`
	Error           error             `json:"error,omitempty"`
	Stack           []string          `json:"stack,omitempty"`
	Memory          []string          `json:"memory,omitempty"`
	Storage         map[string]string `json:"storage,omitempty"`
}

// FormatLogs formats EVM returned structured logs for json output
func FormatLogs(logs []vm.StructLog) []StructLogRes {
	formatted := make([]StructLogRes, len(logs))
	for index, trace := range logs {
		formatted[index] = StructLogRes{
			Pc:              trace.Pc,
			Op:              trace.Op.String(),
			CallerAddress:   trace.CallerAddress,
			ContractAddress: trace.ContractAddress,
			Gas:             trace.Gas,
			GasCost:         trace.GasCost,
			Depth:           trace.Depth,
			Error:           trace.Err,
		}
		if trace.Stack != nil {
			stack := make([]string, len(trace.Stack))
			for i, stackValue := range trace.Stack {
				stack[i] = fmt.Sprintf("%x", math.PaddedBigBytes(stackValue, 32))
			}
			formatted[index].Stack = stack
		}
		if trace.Memory != nil {
			memory := make([]string, 0, (len(trace.Memory)+31)/32)
			for i := 0; i+32 <= len(trace.Memory); i += 32 {
				memory = append(memory, fmt.Sprintf("%x", trace.Memory[i:i+32]))
			}
			formatted[index].Memory = memory
		}
		if trace.Storage != nil {
			storage := make(map[string]string)
			for i, storageValue := range trace.Storage {
				storage[fmt.Sprintf("%x", i)] = fmt.Sprintf("%x", storageValue)
			}
			formatted[index].Storage = storage
		}
	}
	return formatted
}