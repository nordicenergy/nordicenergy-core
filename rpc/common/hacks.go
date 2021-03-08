package common

import (
	"github.com/nordicenergy/nordicenergy-core/consensus/quorum"
	"github.com/nordicenergy/nordicenergy-core/crypto/bls"
	"github.com/nordicenergy/nordicenergy-core/numeric"
)

type setRawStakeHack interface {
	SetRawStake(key bls.SerializedPublicKey, d numeric.Dec)
}

// SetRawStake is a hack, return value is if was successful or not at setting
func SetRawStake(q quorum.Decider, key bls.SerializedPublicKey, d numeric.Dec) bool {
	if setter, ok := q.(setRawStakeHack); ok {
		setter.SetRawStake(key, d)
		return true
	}
	return false
}
