package votepower

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/nordicenergy/nordicenergy-core/crypto/bls"

	shardingconfig "github.com/nordicenergy/nordicenergy-core/internal/configs/sharding"

	"github.com/ethereum/go-ethereum/common"
	bls_core "github.com/nordicenergy/bls/ffi/go/bls"
	"github.com/nordicenergy/nordicenergy-core/numeric"
	"github.com/nordicenergy/nordicenergy-core/shard"
)

var (
	slotList      shard.SlotList
	totalStake    numeric.Dec
	nordicenergyNodes  = 10
	stakedNodes   = 10
	maxAccountGen = int64(98765654323123134)
	accountGen    = rand.New(rand.NewSource(1337))
	maxKeyGen     = int64(98765654323123134)
	keyGen        = rand.New(rand.NewSource(42))
	maxStakeGen   = int64(200)
	stakeGen      = rand.New(rand.NewSource(541))
)

func init() {
	shard.Schedule = shardingconfig.LocalnetSchedule
	for i := 0; i < nordicenergyNodes; i++ {
		newSlot := generateRandomSlot()
		newSlot.EffectiveStake = nil
		slotList = append(slotList, newSlot)
	}

	totalStake = numeric.ZeroDec()
	for j := 0; j < stakedNodes; j++ {
		newSlot := generateRandomSlot()
		totalStake = totalStake.Add(*newSlot.EffectiveStake)
		slotList = append(slotList, newSlot)
	}
}

func generateRandomSlot() shard.Slot {
	addr := common.Address{}
	addr.SetBytes(big.NewInt(int64(accountGen.Int63n(maxAccountGen))).Bytes())
	secretKey := bls_core.SecretKey{}
	secretKey.Deserialize(big.NewInt(int64(keyGen.Int63n(maxKeyGen))).Bytes())
	key := bls.SerializedPublicKey{}
	key.FromLibBLSPublicKey(secretKey.GetPublicKey())
	stake := numeric.NewDecFromBigInt(big.NewInt(int64(stakeGen.Int63n(maxStakeGen))))
	return shard.Slot{addr, key, &stake}
}

func TestCompute(t *testing.T) {
	expectedRoster := NewRoster(shard.BeaconChainShardID)
	// Calculated when generated
	expectedRoster.TotalEffectiveStake = totalStake
	expectedRoster.ngySlotCount = int64(nordicenergyNodes)

	asDecngySlotCount := numeric.NewDec(expectedRoster.ngySlotCount)
	ourPercentage := numeric.ZeroDec()
	theirPercentage := numeric.ZeroDec()

	staked := slotList
	for i := range staked {
		member := AccommodatenordicenergyVote{
			PureStakedVote: PureStakedVote{
				EarningAccount: staked[i].EcdsaAddress,
				Identity:       staked[i].BLSPublicKey,
				GroupPercent:   numeric.ZeroDec(),
				EffectiveStake: numeric.ZeroDec(),
			},
			OverallPercent: numeric.ZeroDec(),
			IsnordicenergyNode:  false,
		}

		// Real Staker
		nordicenergyPercent := shard.Schedule.InstanceForEpoch(big.NewInt(3)).nordicenergyVotePercent()
		externalPercent := shard.Schedule.InstanceForEpoch(big.NewInt(3)).ExternalVotePercent()
		if e := staked[i].EffectiveStake; e != nil {
			member.EffectiveStake = member.EffectiveStake.Add(*e)
			member.GroupPercent = e.Quo(expectedRoster.TotalEffectiveStake)
			member.OverallPercent = member.GroupPercent.Mul(externalPercent)
			theirPercentage = theirPercentage.Add(member.OverallPercent)
		} else { // Our node
			member.IsnordicenergyNode = true
			member.OverallPercent = nordicenergyPercent.Quo(asDecngySlotCount)
			member.GroupPercent = member.OverallPercent.Quo(nordicenergyPercent)
			ourPercentage = ourPercentage.Add(member.OverallPercent)
		}

		expectedRoster.Voters[staked[i].BLSPublicKey] = &member
	}

	expectedRoster.OurVotingPowerTotalPercentage = ourPercentage
	expectedRoster.TheirVotingPowerTotalPercentage = theirPercentage

	computedRoster, err := Compute(&shard.Committee{
		shard.BeaconChainShardID, slotList,
	}, big.NewInt(3))
	if err != nil {
		t.Error("Computed Roster failed on vote summation to net")
	}

	if !compareRosters(expectedRoster, computedRoster, t) {
		t.Errorf("Compute Roster mismatch with expected Roster")
	}
	// Check that voting percents sum to 100
	if !computedRoster.OurVotingPowerTotalPercentage.Add(
		computedRoster.TheirVotingPowerTotalPercentage,
	).Equal(numeric.netDec()) {
		t.Errorf(
			"Total voting power does not equal 1. nordicenergy voting power: %s, Staked voting power: %s",
			computedRoster.OurVotingPowerTotalPercentage,
			computedRoster.TheirVotingPowerTotalPercentage,
		)
	}
}

func compareRosters(a, b *Roster, t *testing.T) bool {
	voterMatch := true
	for k, voter := range a.Voters {
		if other, exists := b.Voters[k]; exists {
			if !compareStakedVoter(voter, other) {
				t.Error("voter slot not match")
				voterMatch = false
			}
		} else {
			t.Error("computed roster missing")
			voterMatch = false
		}
	}
	return a.OurVotingPowerTotalPercentage.Equal(b.OurVotingPowerTotalPercentage) &&
		a.TheirVotingPowerTotalPercentage.Equal(b.TheirVotingPowerTotalPercentage) &&
		a.TotalEffectiveStake.Equal(b.TotalEffectiveStake) &&
		a.ngySlotCount == b.ngySlotCount && voterMatch
}

func compareStakedVoter(a, b *AccommodatenordicenergyVote) bool {
	return a.IsnordicenergyNode == b.IsnordicenergyNode &&
		a.EarningAccount == b.EarningAccount &&
		a.OverallPercent.Equal(b.OverallPercent) &&
		a.EffectiveStake.Equal(b.EffectiveStake)
}
