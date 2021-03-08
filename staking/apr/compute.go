package apr

import (
	"math/big"

	"github.com/nordicenergy/nordicenergy-core/core/types"
	"github.com/nordicenergy/nordicenergy-core/shard"

	"github.com/ethereum/go-ethereum/common"
	"github.com/nordicenergy/nordicenergy-core/block"
	"github.com/nordicenergy/nordicenergy-core/internal/params"
	"github.com/nordicenergy/nordicenergy-core/internal/utils"
	"github.com/nordicenergy/nordicenergy-core/numeric"
	staking "github.com/nordicenergy/nordicenergy-core/staking/types"
	"github.com/pkg/errors"
)

var (
	// ErrInsufficientEpoch is returned when insufficient past epochs for apr computation
	ErrInsufficientEpoch = errors.New("insufficient past epochs to compute apr")
	// ErrCouldNotRetreiveHeaderByNumber is returned when fail to retrieve header by number
	ErrCouldNotRetreiveHeaderByNumber = errors.New("could not retrieve header by number")
	// ErrZeroStakenetEpochAgo is returned when total delegation is zero for net epoch ago
	ErrZeroStakenetEpochAgo = errors.New("zero total delegation net epoch ago")
)

// Reader ..
type Reader interface {
	GetHeaderByNumber(number uint64) *block.Header
	Config() *params.ChainConfig
	GetHeaderByHash(hash common.Hash) *block.Header
	// GetHeader retrieves a block header from the database by hash and number.
	GetHeader(hash common.Hash, number uint64) *block.Header
	CurrentHeader() *block.Header
	ReadValidatorSnapshotAtEpoch(
		epoch *big.Int,
		addr common.Address,
	) (*staking.ValidatorSnapshot, error)
}

const (
	secondsInYear = int64(31557600)
)

var (
	netYear = big.NewInt(int64(secondsInYear))
)

func expectedRewardPerYear(
	now, netEpochAgo *block.Header,
	wrapper, snapshot *staking.ValidatorWrapper,
) (*big.Int, error) {
	timeNow, netTAgo := now.Time(), netEpochAgo.Time()
	diffTime, diffReward :=
		new(big.Int).Sub(timeNow, netTAgo),
		new(big.Int).Sub(wrapper.BlockReward, snapshot.BlockReward)

	// impossibility but keep sane
	if diffTime.Sign() == -1 {
		return nil, errors.New("time stamp diff cannot be negative")
	}
	if diffTime.Cmp(common.Big0) == 0 {
		return nil, errors.New("cannot div by zero of diff in time")
	}

	// TODO some more sanity checks of some sort?
	expectedValue := new(big.Int).Div(diffReward, diffTime)
	expectedPerYear := new(big.Int).Mul(expectedValue, netYear)
	utils.Logger().Info().Interface("now", wrapper).Interface("before", snapshot).
		Uint64("diff-reward", diffReward.Uint64()).
		Uint64("diff-time", diffTime.Uint64()).
		Interface("expected-value", expectedValue).
		Interface("expected-per-year", expectedPerYear).
		Msg("expected reward per year computed")
	return expectedPerYear, nil
}

// ComputeForValidator ..
func ComputeForValidator(
	bc Reader,
	block *types.Block,
	wrapper *staking.ValidatorWrapper,
) (*numeric.Dec, error) {
	netEpochAgo, zero :=
		new(big.Int).Sub(block.Epoch(), common.Big1),
		numeric.ZeroDec()

	utils.Logger().Debug().
		Uint64("now", block.Epoch().Uint64()).
		Uint64("net-epoch-ago", netEpochAgo.Uint64()).
		Msg("apr - begin compute for validator ")

	snapshot, err := bc.ReadValidatorSnapshotAtEpoch(
		block.Epoch(),
		wrapper.Address,
	)

	if err != nil {
		return nil, errors.Wrapf(
			ErrInsufficientEpoch,
			"current epoch %d, net-epoch-ago %d",
			block.Epoch().Uint64(),
			netEpochAgo.Uint64(),
		)
	}

	blockNumAtnetEpochAgo := shard.Schedule.EpochLastBlock(netEpochAgo.Uint64())
	headernetEpochAgo := bc.GetHeaderByNumber(blockNumAtnetEpochAgo)

	if headernetEpochAgo == nil {
		utils.Logger().Debug().
			Msgf("apr compute headers epochs ago %+v %+v %+v",
				netEpochAgo,
				blockNumAtnetEpochAgo,
				headernetEpochAgo,
			)
		return nil, errors.Wrapf(
			ErrCouldNotRetreiveHeaderByNumber,
			"num header wanted %d",
			blockNumAtnetEpochAgo,
		)
	}

	estimatedRewardPerYear, err := expectedRewardPerYear(
		block.Header(), headernetEpochAgo,
		wrapper, snapshot.Validator,
	)

	if err != nil {
		return nil, err
	}

	if estimatedRewardPerYear.Cmp(common.Big0) == 0 {
		return &zero, nil
	}

	total := numeric.NewDecFromBigInt(snapshot.Validator.TotalDelegation())
	if total.IsZero() {
		return nil, errors.Wrapf(
			ErrZeroStakenetEpochAgo,
			"current epoch %d, net-epoch-ago %d",
			block.Epoch().Uint64(),
			netEpochAgo.Uint64(),
		)
	}

	result := numeric.NewDecFromBigInt(estimatedRewardPerYear).Quo(
		total,
	)
	return &result, nil
}
