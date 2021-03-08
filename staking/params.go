package staking

import (
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	isValidatorKeyStr = "nordicenergy/IsValidator/Key/v1"
	isValidatorStr    = "nordicenergy/IsValidator/Value/v1"
	collectRewardsStr = "nordicenergy/CollectRewards"
	delegateStr       = "nordicenergy/Delegate"
)

// keys used to retrieve staking related informatio
var (
	IsValidatorKey      = crypto.Keccak256Hash([]byte(isValidatorKeyStr))
	IsValidator         = crypto.Keccak256Hash([]byte(isValidatorStr))
	CollectRewardsTopic = crypto.Keccak256Hash([]byte(collectRewardsStr))
	DelegateTopic       = crypto.Keccak256Hash([]byte(delegateStr))
)
