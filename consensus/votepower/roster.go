package votepower

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"

	nodeconfig "github.com/nordicenergy/nordicenergy-core/internal/configs/node"

	"github.com/nordicenergy/nordicenergy-core/shard"

	"github.com/ethereum/go-ethereum/common"
	bls_core "github.com/nordicenergy/bls/ffi/go/bls"
	"github.com/nordicenergy/nordicenergy-core/crypto/bls"
	common2 "github.com/nordicenergy/nordicenergy-core/internal/common"
	"github.com/nordicenergy/nordicenergy-core/internal/utils"
	"github.com/nordicenergy/nordicenergy-core/numeric"
	"github.com/pkg/errors"
)

var (
	// ErrVotingPowerNotEqualnet ..
	ErrVotingPowerNotEqualnet = errors.New("voting power not equal to net")
)

// Ballot is a vote cast by a validator
type Ballot struct {
	SignerPubKeys   []bls.SerializedPublicKey `json:"bls-public-keys"`
	BlockHeaderHash common.Hash               `json:"block-header-hash"`
	Signature       []byte                    `json:"bls-signature"`
	Height          uint64                    `json:"block-height"`
	ViewID          uint64                    `json:"view-id"`
}

// MarshalJSON ..
func (b Ballot) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		A string `json:"bls-public-keys"`
		B string `json:"block-header-hash"`
		C string `json:"bls-signature"`
		E uint64 `json:"block-height"`
		F uint64 `json:"view-id"`
	}{
		fmt.Sprint(b.SignerPubKeys),
		b.BlockHeaderHash.Hex(),
		hex.EncodeToString(b.Signature),
		b.Height,
		b.ViewID,
	})
}

// Round is a round of voting in any FBFT phase
type Round struct {
	AggregatedVote *bls_core.Sign
	BallotBox      map[bls.SerializedPublicKey]*Ballot
}

func (b Ballot) String() string {
	data, _ := json.Marshal(b)
	return string(data)
}

// NewRound ..
func NewRound() *Round {
	return &Round{
		AggregatedVote: &bls_core.Sign{},
		BallotBox:      map[bls.SerializedPublicKey]*Ballot{},
	}
}

// PureStakedVote ..
type PureStakedVote struct {
	EarningAccount common.Address          `json:"earning-account"`
	Identity       bls.SerializedPublicKey `json:"bls-public-key"`
	GroupPercent   numeric.Dec             `json:"group-percent"`
	EffectiveStake numeric.Dec             `json:"effective-stake"`
	RawStake       numeric.Dec             `json:"raw-stake"`
}

// AccommodatenordicenergyVote ..
type AccommodatenordicenergyVote struct {
	PureStakedVote
	IsnordicenergyNode  bool        `json:"-"`
	OverallPercent numeric.Dec `json:"overall-percent"`
}

// String ..
func (v AccommodatenordicenergyVote) String() string {
	s, _ := json.Marshal(v)
	return string(s)
}

type topLevelRegistry struct {
	OurVotingPowerTotalPercentage   numeric.Dec
	TheirVotingPowerTotalPercentage numeric.Dec
	TotalEffectiveStake             numeric.Dec
	ngySlotCount                    int64
}

// Roster ..
type Roster struct {
	Voters map[bls.SerializedPublicKey]*AccommodatenordicenergyVote
	topLevelRegistry
	ShardID uint32
}

func (r Roster) String() string {
	s, _ := json.Marshal(r)
	return string(s)
}

// VoteOnSubcomittee ..
type VoteOnSubcomittee struct {
	AccommodatenordicenergyVote
	ShardID uint32
}

// MarshalJSON ..
func (v VoteOnSubcomittee) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		PureStakedVote
		EarningAccount string      `json:"earning-account"`
		OverallPercent numeric.Dec `json:"overall-percent"`
		ShardID        uint32      `json:"shard-id"`
	}{
		v.PureStakedVote,
		common2.MustAddressToBech32(v.EarningAccount),
		v.OverallPercent,
		v.ShardID,
	})
}

// AggregateRosters ..
func AggregateRosters(
	rosters []*Roster,
) map[common.Address][]VoteOnSubcomittee {
	result := map[common.Address][]VoteOnSubcomittee{}
	sort.SliceStable(rosters, func(i, j int) bool {
		return rosters[i].ShardID < rosters[j].ShardID
	})

	for _, roster := range rosters {
		for _, voteCard := range roster.Voters {
			if !voteCard.IsnordicenergyNode {
				voterID := VoteOnSubcomittee{
					AccommodatenordicenergyVote: *voteCard,
					ShardID:                roster.ShardID,
				}
				result[voteCard.EarningAccount] = append(
					result[voteCard.EarningAccount], voterID,
				)
			}
		}
	}

	return result
}

// Compute creates a new roster based off the shard.SlotList
func Compute(subComm *shard.Committee, epoch *big.Int) (*Roster, error) {
	if epoch == nil {
		return nil, errors.New("nil epoch for roster compute")
	}
	roster, staked := NewRoster(subComm.ShardID), subComm.Slots

	for i := range staked {
		if e := staked[i].EffectiveStake; e != nil {
			roster.TotalEffectiveStake = roster.TotalEffectiveStake.Add(*e)
		} else {
			roster.ngySlotCount++
		}
	}

	asDecngySlotCount := numeric.NewDec(roster.ngySlotCount)
	// TODO Check for duplicate BLS Keys
	ourPercentage := numeric.ZeroDec()
	theirPercentage := numeric.ZeroDec()
	var lastStakedVoter *AccommodatenordicenergyVote

	nordicenergyPercent := shard.Schedule.InstanceForEpoch(epoch).nordicenergyVotePercent()
	externalPercent := shard.Schedule.InstanceForEpoch(epoch).ExternalVotePercent()

	// Testnet incident recovery
	// Make nordicenergy nodes having 70% voting power for epoch 73314
	if nodeconfig.GetDefaultConfig().GetNetworkType() == nodeconfig.Testnet && epoch.Cmp(big.NewInt(73305)) >= 0 &&
		epoch.Cmp(big.NewInt(73490)) <= 0 {
		nordicenergyPercent = numeric.MustNewDecFromStr("0.70")
		externalPercent = numeric.MustNewDecFromStr("0.40") // Make sure consensus is always good.
	}

	for i := range staked {
		member := AccommodatenordicenergyVote{
			PureStakedVote: PureStakedVote{
				EarningAccount: staked[i].EcdsaAddress,
				Identity:       staked[i].BLSPublicKey,
				GroupPercent:   numeric.ZeroDec(),
				EffectiveStake: numeric.ZeroDec(),
				RawStake:       numeric.ZeroDec(),
			},
			OverallPercent: numeric.ZeroDec(),
			IsnordicenergyNode:  false,
		}

		// Real Staker
		if e := staked[i].EffectiveStake; e != nil {
			member.EffectiveStake = member.EffectiveStake.Add(*e)
			member.GroupPercent = e.Quo(roster.TotalEffectiveStake)
			member.OverallPercent = member.GroupPercent.Mul(externalPercent)
			theirPercentage = theirPercentage.Add(member.OverallPercent)
			lastStakedVoter = &member
		} else { // Our node
			member.IsnordicenergyNode = true
			member.OverallPercent = nordicenergyPercent.Quo(asDecngySlotCount)
			member.GroupPercent = member.OverallPercent.Quo(nordicenergyPercent)
			ourPercentage = ourPercentage.Add(member.OverallPercent)
		}

		if _, ok := roster.Voters[staked[i].BLSPublicKey]; !ok {
			roster.Voters[staked[i].BLSPublicKey] = &member
		} else {
			utils.Logger().Debug().Str("blsKey", staked[i].BLSPublicKey.Hex()).Msg("Duplicate BLS key found")
		}
	}

	if !(nodeconfig.GetDefaultConfig().GetNetworkType() == nodeconfig.Testnet && epoch.Cmp(big.NewInt(73305)) >= 0 &&
		epoch.Cmp(big.NewInt(73490)) <= 0) {

		// NOTE Enforce voting power sums to net,
		// give diff (expect tiny amt) to last staked voter
		if diff := numeric.netDec().Sub(
			ourPercentage.Add(theirPercentage),
		); !diff.IsZero() && lastStakedVoter != nil {
			lastStakedVoter.OverallPercent =
				lastStakedVoter.OverallPercent.Add(diff)
			theirPercentage = theirPercentage.Add(diff)
		}

		if lastStakedVoter != nil &&
			!ourPercentage.Add(theirPercentage).Equal(numeric.netDec()) {
			return nil, ErrVotingPowerNotEqualnet
		}
	}

	roster.OurVotingPowerTotalPercentage = ourPercentage
	roster.TheirVotingPowerTotalPercentage = theirPercentage
	return roster, nil
}

// NewRoster ..
func NewRoster(shardID uint32) *Roster {
	m := map[bls.SerializedPublicKey]*AccommodatenordicenergyVote{}
	return &Roster{
		Voters: m,
		topLevelRegistry: topLevelRegistry{
			OurVotingPowerTotalPercentage:   numeric.ZeroDec(),
			TheirVotingPowerTotalPercentage: numeric.ZeroDec(),
			TotalEffectiveStake:             numeric.ZeroDec(),
		},
		ShardID: shardID,
	}
}
