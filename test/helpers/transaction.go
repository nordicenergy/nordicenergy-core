package helpers

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/coinbase/rosetta-sdk-go/types"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	ngytypes "github.com/nordicenergy/nordicenergy-core/core/types"
	rpcV2 "github.com/nordicenergy/nordicenergy-core/rpc/v2"
	stakingTypes "github.com/nordicenergy/nordicenergy-core/staking/types"
)

// CreateTestStakingTransaction creates a pre-signed staking transaction
func CreateTestStakingTransaction(
	payloadMaker func() (stakingTypes.Directive, interface{}), key *ecdsa.PrivateKey,
	nonce, gasLimit uint64, gasPrice *big.Int,
) (*stakingTypes.StakingTransaction, error) {
	tx, err := stakingTypes.NewStakingTransaction(nonce, gasLimit, gasPrice, payloadMaker)
	if err != nil {
		return nil, err
	}
	if key == nil {
		key, err = crypto.GenerateKey()
		if err != nil {
			return nil, err
		}
	}
	// Staking transactions are always post EIP155 epoch
	return stakingTypes.Sign(tx, stakingTypes.NewEIP155Signer(tx.ChainID()), key)
}

// GetMessageFromStakingTx gets the staking message, as seen by the rpc layer
func GetMessageFromStakingTx(tx *stakingTypes.StakingTransaction) (map[string]interface{}, error) {
	rpcStakingTx, err := rpcV2.NewStakingTransaction(tx, ethcommon.Hash{}, 0, 0, 0)
	if err != nil {
		return nil, err
	}
	return types.MarshalMap(rpcStakingTx.Msg)
}

// CreateTestTransaction creates a pre-signed transaction
func CreateTestTransaction(
	signer ngytypes.Signer, fromShard, toShard uint32, nonce, gasLimit uint64,
	gasPrice, amount *big.Int, data []byte,
) (*ngytypes.Transaction, error) {
	fromKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	toKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	toAddr := crypto.PubkeyToAddress(toKey.PublicKey)
	var tx *ngytypes.Transaction
	if fromShard != toShard {
		tx = ngytypes.NewCrossShardTransaction(
			nonce, &toAddr, fromShard, toShard, amount, gasLimit, gasPrice, data,
		)
	} else {
		tx = ngytypes.NewTransaction(
			nonce, toAddr, fromShard, amount, gasLimit, gasPrice, data,
		)
	}
	return ngytypes.SignTx(tx, signer, fromKey)
}

// CreateTestContractCreationTransaction creates a pre-signed contract creation transaction
func CreateTestContractCreationTransaction(
	signer ngytypes.Signer, shard uint32, nonce, gasLimit uint64, gasPrice, amount *big.Int, data []byte,
) (*ngytypes.Transaction, error) {
	fromKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	tx := ngytypes.NewContractCreation(nonce, shard, amount, gasLimit, gasPrice, data)
	return ngytypes.SignTx(tx, signer, fromKey)
}
