package ngy

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/nordicenergy/nordicenergy-core/core/types"
)

// GetPoolStats returns the number of pending and queued transactions
func (ngy *nordicenergy) GetPoolStats() (pendingCount, queuedCount int) {
	return ngy.TxPool.Stats()
}

// GetPoolNonce ...
func (ngy *nordicenergy) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return ngy.TxPool.State().GetNonce(addr), nil
}

// GetPoolTransaction ...
func (ngy *nordicenergy) GetPoolTransaction(hash common.Hash) types.PoolTransaction {
	return ngy.TxPool.Get(hash)
}

// GetPendingCXReceipts ..
func (ngy *nordicenergy) GetPendingCXReceipts() []*types.CXReceiptsProof {
	return ngy.NodeAPI.PendingCXReceipts()
}

// GetPoolTransactions returns pool transactions.
func (ngy *nordicenergy) GetPoolTransactions() (types.PoolTransactions, error) {
	pending, err := ngy.TxPool.Pending()
	if err != nil {
		return nil, err
	}
	queued, err := ngy.TxPool.Queued()
	if err != nil {
		return nil, err
	}
	var txs types.PoolTransactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	for _, batch := range queued {
		txs = append(txs, batch...)
	}
	return txs, nil
}
