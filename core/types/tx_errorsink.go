package types

import (
	"time"

	lru "github.com/hashicorp/golang-lru"

	"github.com/nordicenergy/nordicenergy-core/internal/utils"
	staking "github.com/nordicenergy/nordicenergy-core/staking/types"
)

const (
	plainTxSinkLimit   = 1024
	stakingTxSinkLimit = 1024
	logTag             = "[TransactinetrrorSink]"
)

// TransactinetrrorReport ..
type TransactinetrrorReport struct {
	TxHashID             string `json:"tx-hash-id"`
	StakingDirective     string `json:"directive-kind,omitempty"`
	TimestampOfRejection int64  `json:"time-at-rejection"`
	ErrMessage           string `json:"error-message"`
}

// TransactinetrrorReports ..
type TransactinetrrorReports []*TransactinetrrorReport

// TransactinetrrorSink is where all failed transactions get reported.
// Note that the keys of the lru caches are tx-hash strings.
type TransactinetrrorSink struct {
	failedPlainTxs   *lru.Cache
	failedStakingTxs *lru.Cache
}

// NewTransactinetrrorSink ..
func NewTransactinetrrorSink() *TransactinetrrorSink {
	failedPlainTx, _ := lru.New(plainTxSinkLimit)
	failedStakingTx, _ := lru.New(stakingTxSinkLimit)
	return &TransactinetrrorSink{
		failedPlainTxs:   failedPlainTx,
		failedStakingTxs: failedStakingTx,
	}
}

// Add a transaction to the error sink with the given error
func (sink *TransactinetrrorSink) Add(tx PoolTransaction, err error) {
	// no-op if no error is provided
	if err == nil {
		return
	}
	if plainTx, ok := tx.(*Transaction); ok {
		hash := plainTx.Hash().String()
		sink.failedPlainTxs.Add(hash, &TransactinetrrorReport{
			TxHashID:             hash,
			TimestampOfRejection: time.Now().Unix(),
			ErrMessage:           err.Error(),
		})
		utils.Logger().Debug().
			Str("tag", logTag).
			Interface("tx-hash-id", hash).
			Msgf("Added plain transaction error message")
	} else if ethTx, ok := tx.(*EthTransaction); ok {
		hash := ethTx.Hash().String()
		sink.failedPlainTxs.Add(hash, &TransactinetrrorReport{
			TxHashID:             hash,
			TimestampOfRejection: time.Now().Unix(),
			ErrMessage:           err.Error(),
		})
		utils.Logger().Debug().
			Str("tag", logTag).
			Interface("tx-hash-id", hash).
			Msgf("Added eth transaction error message")
	} else if stakingTx, ok := tx.(*staking.StakingTransaction); ok {
		hash := stakingTx.Hash().String()
		sink.failedStakingTxs.Add(hash, &TransactinetrrorReport{
			TxHashID:             hash,
			StakingDirective:     stakingTx.StakingType().String(),
			TimestampOfRejection: time.Now().Unix(),
			ErrMessage:           err.Error(),
		})
		utils.Logger().Debug().
			Str("tag", logTag).
			Interface("tx-hash-id", hash).
			Msgf("Added staking transaction error message")
	} else {
		utils.Logger().Error().
			Str("tag", logTag).
			Interface("tx", tx).
			Msg("Attempted to add an unknown transaction type")
	}
}

// Contains checks if there is an error associated with the given hash
// Note that the keys of the lru caches are tx-hash strings.
func (sink *TransactinetrrorSink) Contains(hash string) bool {
	return sink.failedPlainTxs.Contains(hash) || sink.failedStakingTxs.Contains(hash)
}

// Remove a transaction's error from the error sink
func (sink *TransactinetrrorSink) Remove(tx PoolTransaction) {
	if plainTx, ok := tx.(*Transaction); ok {
		hash := plainTx.Hash().String()
		sink.failedPlainTxs.Remove(hash)
		utils.Logger().Debug().
			Str("tag", logTag).
			Interface("tx-hash-id", hash).
			Msgf("Removed plain transaction error message")
	} else if ethTx, ok := tx.(*EthTransaction); ok {
		hash := ethTx.Hash().String()
		sink.failedPlainTxs.Remove(hash)
		utils.Logger().Debug().
			Str("tag", logTag).
			Interface("tx-hash-id", hash).
			Msgf("Removed plain transaction error message")
	} else if stakingTx, ok := tx.(*staking.StakingTransaction); ok {
		hash := stakingTx.Hash().String()
		sink.failedStakingTxs.Remove(hash)
		utils.Logger().Debug().
			Str("tag", logTag).
			Interface("tx-hash-id", hash).
			Msgf("Removed staking transaction error message")
	} else {
		utils.Logger().Error().
			Str("tag", logTag).
			Interface("tx", tx).
			Msg("Attempted to remove an unknown transaction type")
	}
}

// PlainReport ..
func (sink *TransactinetrrorSink) PlainReport() TransactinetrrorReports {
	return reportErrorsFromLruCache(sink.failedPlainTxs)
}

// StakingReport ..
func (sink *TransactinetrrorSink) StakingReport() TransactinetrrorReports {
	return reportErrorsFromLruCache(sink.failedStakingTxs)
}

// PlainCount ..
func (sink *TransactinetrrorSink) PlainCount() int {
	return sink.failedPlainTxs.Len()
}

// StakingCount ..
func (sink *TransactinetrrorSink) StakingCount() int {
	return sink.failedStakingTxs.Len()
}

// reportErrorsFromLruCache is a helper for reporting errors
// from the TransactinetrrorSink's lru cache. Do not use this function directly,
// use the respective public methods of TransactinetrrorSink.
func reportErrorsFromLruCache(lruCache *lru.Cache) TransactinetrrorReports {
	rpcErrors := TransactinetrrorReports{}
	for _, txHash := range lruCache.Keys() {
		rpcErrorFetch, ok := lruCache.Get(txHash)
		if !ok {
			utils.Logger().Warn().
				Str("tag", logTag).
				Interface("tx-hash-id", txHash).
				Msgf("Error not found in sink")
			continue
		}
		rpcError, ok := rpcErrorFetch.(*TransactinetrrorReport)
		if !ok {
			utils.Logger().Error().
				Str("tag", logTag).
				Interface("tx-hash-id", txHash).
				Msgf("Invalid type of value in sink")
			continue
		}
		rpcErrors = append(rpcErrors, rpcError)
	}
	return rpcErrors
}
