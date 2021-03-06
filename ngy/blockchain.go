package ngy

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/bloombits"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/nordicenergy/nordicenergy-core/block"
	"github.com/nordicenergy/nordicenergy-core/core"
	"github.com/nordicenergy/nordicenergy-core/core/rawdb"
	"github.com/nordicenergy/nordicenergy-core/core/state"
	"github.com/nordicenergy/nordicenergy-core/core/types"
	"github.com/nordicenergy/nordicenergy-core/crypto/bls"
	internal_bls "github.com/nordicenergy/nordicenergy-core/crypto/bls"
	internal_common "github.com/nordicenergy/nordicenergy-core/internal/common"
	"github.com/nordicenergy/nordicenergy-core/internal/params"
	"github.com/nordicenergy/nordicenergy-core/internal/utils"
	"github.com/nordicenergy/nordicenergy-core/shard"
	"github.com/nordicenergy/nordicenergy-core/staking/availability"
	stakingReward "github.com/nordicenergy/nordicenergy-core/staking/reward"
	"github.com/pkg/errors"
)

// ChainConfig ...
func (ngy *nordicenergy) ChainConfig() *params.ChainConfig {
	return ngy.BlockChain.Config()
}

// GetShardState ...
func (ngy *nordicenergy) GetShardState() (*shard.State, error) {
	return ngy.BlockChain.ReadShardState(ngy.BlockChain.CurrentHeader().Epoch())
}

// GetBlockSigners ..
func (ngy *nordicenergy) GetBlockSigners(
	ctx context.Context, blockNum rpc.BlockNumber,
) (shard.SlotList, *internal_bls.Mask, error) {
	blk, err := ngy.BlockByNumber(ctx, blockNum)
	if err != nil {
		return nil, nil, err
	}
	blockWithSigners, err := ngy.BlockByNumber(ctx, blockNum+1)
	if err != nil {
		return nil, nil, err
	}
	if blockWithSigners == nil {
		return nil, nil, fmt.Errorf("block number %v not found", blockNum+1)
	}
	committee, err := ngy.GetValidators(blk.Epoch())
	if err != nil {
		return nil, nil, err
	}
	pubKeys := make([]internal_bls.PublicKeyWrapper, len(committee.Slots))
	for i, validator := range committee.Slots {
		key, err := bls.BytesToBLSPublicKey(validator.BLSPublicKey[:])
		if err != nil {
			return nil, nil, err
		}
		pubKeys[i] = internal_bls.PublicKeyWrapper{
			Bytes:  validator.BLSPublicKey,
			Object: key,
		}
	}
	mask, err := internal_bls.NewMask(pubKeys, nil)
	if err != nil {
		return nil, nil, err
	}
	err = mask.SetMask(blockWithSigners.Header().LastCommitBitmap())
	if err != nil {
		return nil, nil, err
	}
	return committee.Slots, mask, nil
}

// DetailedBlockSignerInfo contains all of the block singing information
type DetailedBlockSignerInfo struct {
	// Signers are all the signers for the block
	Signers shard.SlotList
	// Committee when the block was signed.
	Committee shard.SlotList
	BlockHash common.Hash
}

// GetDetailedBlockSignerInfo fetches the block signer information for any non-genesis block
func (ngy *nordicenergy) GetDetailedBlockSignerInfo(
	ctx context.Context, blk *types.Block,
) (*DetailedBlockSignerInfo, error) {
	parentBlk, err := ngy.BlockByNumber(ctx, rpc.BlockNumber(blk.NumberU64()-1))
	if err != nil {
		return nil, err
	}
	parentShardState, err := ngy.BlockChain.ReadShardState(parentBlk.Epoch())
	if err != nil {
		return nil, err
	}
	committee, signers, _, err := availability.BallotResult(
		parentBlk.Header(), blk.Header(), parentShardState, blk.ShardID(),
	)
	return &DetailedBlockSignerInfo{
		Signers:   signers,
		Committee: committee,
		BlockHash: blk.Hash(),
	}, nil
}

// PreStakingBlockRewards are the rewards for a block in the pre-staking era (epoch < staking epoch).
type PreStakingBlockRewards map[common.Address]*big.Int

// GetPreStakingBlockRewards for the given block number.
// Calculated rewards are dnet exactly like chain.AccumulateRewardsAndCountSigs.
func (ngy *nordicenergy) GetPreStakingBlockRewards(
	ctx context.Context, blk *types.Block,
) (PreStakingBlockRewards, error) {
	if ngy.IsStakingEpoch(blk.Epoch()) {
		return nil, fmt.Errorf("block %v is in staking era", blk.Number())
	}

	if cachedReward, ok := ngy.preStakingBlockRewardsCache.Get(blk.Hash()); ok {
		return cachedReward.(PreStakingBlockRewards), nil
	}
	rewards := PreStakingBlockRewards{}

	sigInfo, err := ngy.GetDetailedBlockSignerInfo(ctx, blk)
	if err != nil {
		return nil, err
	}
	last := big.NewInt(0)
	count := big.NewInt(int64(len(sigInfo.Signers)))
	for i, slot := range sigInfo.Signers {
		rewardsForThisAddr, ok := rewards[slot.EcdsaAddress]
		if !ok {
			rewardsForThisAddr = big.NewInt(0)
		}
		cur := big.NewInt(0)
		cur.Mul(stakingReward.PreStakedBlocks, big.NewInt(int64(i+1))).Div(cur, count)
		reward := big.NewInt(0).Sub(cur, last)
		rewards[slot.EcdsaAddress] = new(big.Int).Add(reward, rewardsForThisAddr)
		last = cur
	}

	// Report tx fees of the coinbase (== leader)
	receipts, err := ngy.GetReceipts(ctx, blk.Hash())
	if err != nil {
		return nil, err
	}
	txFees := big.NewInt(0)
	for _, tx := range blk.Transactions() {
		txnHash := tx.HashByType()
		dbTx, _, _, receiptIndex := rawdb.ReadTransaction(ngy.ChainDb(), txnHash)
		if dbTx == nil {
			return nil, fmt.Errorf("could not find receipt for tx: %v", txnHash.String())
		}
		if len(receipts) <= int(receiptIndex) {
			return nil, fmt.Errorf("invalid receipt indext %v (>= num receipts: %v) for tx: %v",
				receiptIndex, len(receipts), txnHash.String())
		}
		txFee := new(big.Int).Mul(tx.GasPrice(), big.NewInt(int64(receipts[receiptIndex].GasUsed)))
		txFees = new(big.Int).Add(txFee, txFees)
	}
	for _, stx := range blk.StakingTransactions() {
		dbsTx, _, _, receiptIndex := rawdb.ReadStakingTransaction(ngy.ChainDb(), stx.Hash())
		if dbsTx == nil {
			return nil, fmt.Errorf("could not find receipt for tx: %v", stx.Hash().String())
		}
		if len(receipts) <= int(receiptIndex) {
			return nil, fmt.Errorf("invalid receipt indext %v (>= num receipts: %v) for tx: %v",
				receiptIndex, len(receipts), stx.Hash().String())
		}
		txFee := new(big.Int).Mul(stx.GasPrice(), big.NewInt(int64(receipts[receiptIndex].GasUsed)))
		txFees = new(big.Int).Add(txFee, txFees)
	}
	if amt, ok := rewards[blk.Header().Coinbase()]; ok {
		rewards[blk.Header().Coinbase()] = new(big.Int).Add(amt, txFees)
	} else {
		rewards[blk.Header().Coinbase()] = txFees
	}

	ngy.preStakingBlockRewardsCache.Add(blk.Hash(), rewards)
	return rewards, nil
}

// GetLatestChainHeaders ..
func (ngy *nordicenergy) GetLatestChainHeaders() *block.HeaderPair {
	return &block.HeaderPair{
		BeaconHeader: ngy.BeaconChain.CurrentHeader(),
		ShardHeader:  ngy.BlockChain.CurrentHeader(),
	}
}

// GetLastCrossLinks ..
func (ngy *nordicenergy) GetLastCrossLinks() ([]*types.CrossLink, error) {
	crossLinks := []*types.CrossLink{}
	for i := uint32(1); i < shard.Schedule.InstanceForEpoch(ngy.CurrentBlock().Epoch()).NumShards(); i++ {
		link, err := ngy.BlockChain.ReadShardLastCrossLink(i)
		if err != nil {
			return nil, err
		}
		crossLinks = append(crossLinks, link)
	}

	return crossLinks, nil
}

// CurrentBlock ...
func (ngy *nordicenergy) CurrentBlock() *types.Block {
	return types.NewBlockWithHeader(ngy.BlockChain.CurrentHeader())
}

// GetBlock ...
func (ngy *nordicenergy) GetBlock(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return ngy.BlockChain.GetBlockByHash(hash), nil
}

// GetCurrentBadBlocks ..
func (ngy *nordicenergy) GetCurrentBadBlocks() []core.BadBlock {
	return ngy.BlockChain.BadBlocks()
}

// GetBalance returns balance of an given address.
func (ngy *nordicenergy) GetBalance(ctx context.Context, address common.Address, blockNum rpc.BlockNumber) (*big.Int, error) {
	s, _, err := ngy.StateAndHeaderByNumber(ctx, blockNum)
	if s == nil || err != nil {
		return nil, err
	}
	return s.GetBalance(address), s.Error()
}

// BlockByNumber ...
func (ngy *nordicenergy) BlockByNumber(ctx context.Context, blockNum rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if blockNum == rpc.PendingBlockNumber {
		return nil, errors.New("not implemented")
	}
	// Otherwise resolve and return the block
	if blockNum == rpc.LatestBlockNumber {
		return ngy.BlockChain.CurrentBlock(), nil
	}
	return ngy.BlockChain.GetBlockByNumber(uint64(blockNum)), nil
}

// HeaderByNumber ...
func (ngy *nordicenergy) HeaderByNumber(ctx context.Context, blockNum rpc.BlockNumber) (*block.Header, error) {
	// Pending block is only known by the miner
	if blockNum == rpc.PendingBlockNumber {
		return nil, errors.New("not implemented")
	}
	// Otherwise resolve and return the block
	if blockNum == rpc.LatestBlockNumber {
		return ngy.BlockChain.CurrentBlock().Header(), nil
	}
	return ngy.BlockChain.GetHeaderByNumber(uint64(blockNum)), nil
}

// HeaderByHash ...
func (ngy *nordicenergy) HeaderByHash(ctx context.Context, blockHash common.Hash) (*block.Header, error) {
	header := ngy.BlockChain.GetHeaderByHash(blockHash)
	if header == nil {
		return nil, errors.New("Header is not found")
	}
	return header, nil
}

// StateAndHeaderByNumber ...
func (ngy *nordicenergy) StateAndHeaderByNumber(ctx context.Context, blockNum rpc.BlockNumber) (*state.DB, *block.Header, error) {
	// Pending state is only known by the miner
	if blockNum == rpc.PendingBlockNumber {
		return nil, nil, errors.New("not implemented")
	}
	// Otherwise resolve the block number and return its state
	header, err := ngy.HeaderByNumber(ctx, blockNum)
	if header == nil || err != nil {
		return nil, nil, err
	}
	stateDb, err := ngy.BlockChain.StateAt(header.Root())
	return stateDb, header, err
}

// GetLeaderAddress returns the net address of the leader, given the coinbaseAddr.
// Note that the coinbaseAddr is overloaded with the BLS pub key hash in staking era.
func (ngy *nordicenergy) GetLeaderAddress(coinbaseAddr common.Address, epoch *big.Int) string {
	if ngy.IsStakingEpoch(epoch) {
		if leader, exists := ngy.leaderCache.Get(coinbaseAddr); exists {
			bech32, _ := internal_common.AddressToBech32(leader.(common.Address))
			return bech32
		}
		committee, err := ngy.GetValidators(epoch)
		if err != nil {
			return ""
		}
		for _, val := range committee.Slots {
			addr := utils.GetAddressFromBLSPubKeyBytes(val.BLSPublicKey[:])
			ngy.leaderCache.Add(addr, val.EcdsaAddress)
			if addr == coinbaseAddr {
				bech32, _ := internal_common.AddressToBech32(val.EcdsaAddress)
				return bech32
			}
		}
		return "" // Did not find matching address
	}
	bech32, _ := internal_common.AddressToBech32(coinbaseAddr)
	return bech32
}

// Filter related APIs

// GetLogs ...
func (ngy *nordicenergy) GetLogs(ctx context.Context, blockHash common.Hash) ([][]*types.Log, error) {
	receipts := ngy.BlockChain.GetReceiptsByHash(blockHash)
	if receipts == nil {
		return nil, errors.New("Missing receipts")
	}
	logs := make([][]*types.Log, len(receipts))
	for i, receipt := range receipts {
		logs[i] = receipt.Logs
	}
	return logs, nil
}

// ServiceFilter ...
func (ngy *nordicenergy) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	// TODO(dm): implement
}

// SubscribeNewTxsEvent subscribes new tx event.
// TODO: this is not implemented or verified yet for nordicenergy.
func (ngy *nordicenergy) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return ngy.TxPool.SubscribeNewTxsEvent(ch)
}

// SubscribeChainEvent subscribes chain event.
// TODO: this is not implemented or verified yet for nordicenergy.
func (ngy *nordicenergy) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return ngy.BlockChain.SubscribeChainEvent(ch)
}

// SubscribeChainHeadEvent subcribes chain head event.
// TODO: this is not implemented or verified yet for nordicenergy.
func (ngy *nordicenergy) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return ngy.BlockChain.SubscribeChainHeadEvent(ch)
}

// SubscribeChainSideEvent subcribes chain side event.
// TODO: this is not implemented or verified yet for nordicenergy.
func (ngy *nordicenergy) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return ngy.BlockChain.SubscribeChainSideEvent(ch)
}

// SubscribeRemovedLogsEvent subcribes removed logs event.
// TODO: this is not implemented or verified yet for nordicenergy.
func (ngy *nordicenergy) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return ngy.BlockChain.SubscribeRemovedLogsEvent(ch)
}

// SubscribeLogsEvent subcribes log event.
// TODO: this is not implemented or verified yet for nordicenergy.
func (ngy *nordicenergy) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return ngy.BlockChain.SubscribeLogsEvent(ch)
}
