package rpc

import (
	"context"
	"fmt"
	"math/big"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/nordicenergy/nordicenergy-core/ngy"
	internal_common "github.com/nordicenergy/nordicenergy-core/internal/common"
	"github.com/nordicenergy/nordicenergy-core/shard"
)

const (
	validatorsPageSize = 100
)

// PublicStakingService provides an API to access nordicenergy's staking services.
// It offers only methods that operate on public data that is freely available to anynet.
type PublicStakingService struct {
	ngy     *ngy.nordicenergy
	version Version
}

// NewPublicStakingAPI creates a new API for the RPC interface
func NewPublicStakingAPI(ngy *ngy.nordicenergy, version Version) rpc.API {
	return rpc.API{
		Namespace: version.Namespace(),
		Version:   APIVersion,
		Service:   &PublicStakingService{ngy, version},
		Public:    true,
	}
}

// getBalanceByBlockNumber returns balance by block number at given eth blockNum without checks
func (s *PublicStakingService) getBalanceByBlockNumber(
	ctx context.Context, address string, blockNum rpc.BlockNumber,
) (*big.Int, error) {
	addr := internal_common.ParseAddr(address)
	balance, err := s.ngy.GetBalance(ctx, addr, blockNum)
	if err != nil {
		return nil, err
	}
	return balance, nil
}

// getAllValidatorInformation helper function to get all validator information for a given eth block number
func (s *PublicStakingService) getAllValidatorInformation(
	ctx context.Context, page int, blockNum rpc.BlockNumber,
) ([]StructuredResponse, error) {
	if page < -1 {
		return nil, errors.Errorf("page given %d cannot be less than -1", page)
	}

	// Get all validators
	addresses := s.ngy.GetAllValidatorAddresses()
	if page != -1 && len(addresses) <= page*validatorsPageSize {
		return []StructuredResponse{}, nil
	}

	// Set page start
	validatorsNum := len(addresses)
	start := 0
	if page != -1 {
		validatorsNum = validatorsPageSize
		start = page * validatorsPageSize
		if len(addresses)-start < validatorsPageSize {
			validatorsNum = len(addresses) - start
		}
	}

	// Fetch block
	blk, err := s.ngy.BlockByNumber(ctx, blockNum)
	if err != nil {
		return nil, errors.Wrapf(err, "could not retrieve the blk information for blk number: %d", blockNum)
	}

	// Fetch validator information for block
	validators := []StructuredResponse{}
	for i := start; i < start+validatorsNum; i++ {
		validatorInfo, err := s.ngy.GetValidatorInformation(addresses[i], blk)
		if err == nil {
			// Response output is the same for all versions
			information, err := NewStructuredResponse(validatorInfo)
			if err != nil {
				return nil, err
			}
			validators = append(validators, information)
		}
	}
	return validators, nil
}

// GetTotalStaking returns total staking by validators, only meant to be called on beaconchain
// explorer node
func (s *PublicStakingService) GetTotalStaking(
	ctx context.Context,
) (*big.Int, error) {
	if !isBeaconShard(s.ngy) {
		return nil, ErrNotBeaconShard
	}

	// Response output is the same for all versions
	return s.ngy.GetTotalStakingSnapshot(), nil
}

// GetMedianRawStakeSnapshot returns the raw median stake, only meant to be called on beaconchain
// explorer node
func (s *PublicStakingService) GetMedianRawStakeSnapshot(
	ctx context.Context,
) (StructuredResponse, error) {
	if !isBeaconShard(s.ngy) {
		return nil, ErrNotBeaconShard
	}

	// Fetch snapshot
	snapshot, err := s.ngy.GetMedianRawStakeSnapshot()
	if err != nil {
		return nil, err
	}

	// Response output is the same for all versions
	return NewStructuredResponse(snapshot)
}

// GetElectedValidatorAddresses returns elected validator addresses.
func (s *PublicStakingService) GetElectedValidatorAddresses(
	ctx context.Context,
) ([]string, error) {
	if !isBeaconShard(s.ngy) {
		return nil, ErrNotBeaconShard
	}

	// Fetch elected validators
	electedAddresses := s.ngy.GetElectedValidatorAddresses()
	addresses := make([]string, len(electedAddresses))
	for i, addr := range electedAddresses {
		netAddr, _ := internal_common.AddressToBech32(addr)
		// Response output is the same for all versions
		addresses[i] = netAddr
	}
	return addresses, nil
}

// GetValidators returns validators list for a particular epoch.
func (s *PublicStakingService) GetValidators(
	ctx context.Context, epoch int64,
) (StructuredResponse, error) {
	// Fetch the Committee
	cmt, err := s.ngy.GetValidators(big.NewInt(epoch))
	if err != nil {
		return nil, err
	}
	balanceQueryBlock := shard.Schedule.EpochLastBlock(uint64(epoch))
	if balanceQueryBlock > s.ngy.CurrentBlock().NumberU64() {
		balanceQueryBlock = s.ngy.CurrentBlock().NumberU64()
	}

	validators := []StructuredResponse{}
	for _, validator := range cmt.Slots {
		// Fetch the balance of the validator
		netAddress, err := internal_common.AddressToBech32(validator.EcdsaAddress)
		if err != nil {
			return nil, err
		}
		validatorBalance, err := s.getBalanceByBlockNumber(ctx, netAddress, rpc.BlockNumber(balanceQueryBlock))
		if err != nil {
			return nil, err
		}

		// Format the response according to the version
		var validatorsFields StructuredResponse
		switch s.version {
		case V1:
			validatorsFields = StructuredResponse{
				"address": netAddress,
				"balance": (*hexutil.Big)(validatorBalance),
			}
		case V2:
			validatorsFields = StructuredResponse{
				"address": netAddress,
				"balance": validatorBalance,
			}
		default:
			return nil, ErrUnknownRPCVersion
		}
		validators = append(validators, validatorsFields)
	}
	result := StructuredResponse{
		"shardID":    cmt.ShardID,
		"validators": validators,
	}
	return result, nil
}

// GetAllValidatorAddresses returns all validator addresses.
func (s *PublicStakingService) GetAllValidatorAddresses(
	ctx context.Context,
) ([]string, error) {
	if !isBeaconShard(s.ngy) {
		return nil, ErrNotBeaconShard
	}

	// Fetch all validator addresses
	validatorAddresses := s.ngy.GetAllValidatorAddresses()
	addresses := make([]string, len(validatorAddresses))
	for i, addr := range validatorAddresses {
		netAddr, _ := internal_common.AddressToBech32(addr)
		// Response output is the same for all versions
		addresses[i] = netAddr
	}
	return addresses, nil
}

// GetValidatorKeys returns list of bls public keys in the committee for a particular epoch.
func (s *PublicStakingService) GetValidatorKeys(
	ctx context.Context, epoch int64,
) ([]string, error) {
	// Fetch the Committee
	cmt, err := s.ngy.GetValidators(big.NewInt(epoch))
	if err != nil {
		return nil, err
	}

	// Response output is the same for all versions
	validators := make([]string, len(cmt.Slots))
	for i, v := range cmt.Slots {
		validators[i] = v.BLSPublicKey.Hex()
	}
	return validators, nil
}

// GetAllValidatorInformation returns information about all validators.
// If page is -1, return all instead of `validatorsPageSize` elements.
func (s *PublicStakingService) GetAllValidatorInformation(
	ctx context.Context, page int,
) ([]StructuredResponse, error) {
	if !isBeaconShard(s.ngy) {
		return nil, ErrNotBeaconShard
	}

	// fetch current block number
	blockNum := s.ngy.CurrentBlock().NumberU64()

	// delete cache for previous block
	prevKey := fmt.Sprintf("all-info-%d", blockNum-1)
	s.ngy.SingleFlightForgetKey(prevKey)

	// Fetch all validator information in a single flight request
	key := fmt.Sprintf("all-info-%d", blockNum)
	res, err := s.ngy.SingleFlightRequest(
		key,
		func() (interface{}, error) {
			return s.getAllValidatorInformation(ctx, page, rpc.LatestBlockNumber)
		},
	)
	if err != nil {
		return nil, err
	}

	// Response output is the same for all versions
	return res.([]StructuredResponse), nil
}

// GetAllValidatorInformationByBlockNumber returns information about all validators.
// If page is -1, return all instead of `validatorsPageSize` elements.
func (s *PublicStakingService) GetAllValidatorInformationByBlockNumber(
	ctx context.Context, page int, blockNumber BlockNumber,
) ([]StructuredResponse, error) {
	// Process number based on version
	blockNum := blockNumber.EthBlockNumber()

	if !isBeaconShard(s.ngy) {
		return nil, ErrNotBeaconShard
	}
	if isBlockGreaterThanLatest(s.ngy, blockNum) {
		return nil, ErrRequestedBlockTooHigh
	}

	// Response output is the same for all versions
	return s.getAllValidatorInformation(ctx, page, blockNum)
}

// GetValidatorInformation returns information about a validator.
func (s *PublicStakingService) GetValidatorInformation(
	ctx context.Context, address string,
) (StructuredResponse, error) {
	if !isBeaconShard(s.ngy) {
		return nil, ErrNotBeaconShard
	}

	// Fetch latest block
	blk, err := s.ngy.BlockByNumber(ctx, rpc.LatestBlockNumber)
	if err != nil {
		return nil, errors.Wrapf(err, "could not retrieve the latest blk information")
	}

	// Fetch validator information
	validatorInfo, err := s.ngy.GetValidatorInformation(internal_common.ParseAddr(address), blk)
	if err != nil {
		return nil, err
	}

	// Response output is the same for all versions
	return NewStructuredResponse(validatorInfo)
}

// GetValidatorInformationByBlockNumber returns information about a validator.
func (s *PublicStakingService) GetValidatorInformationByBlockNumber(
	ctx context.Context, address string, blockNumber BlockNumber,
) (StructuredResponse, error) {
	// Process number based on version
	blockNum := blockNumber.EthBlockNumber()

	// Fetch block
	if !isBeaconShard(s.ngy) {
		return nil, ErrNotBeaconShard
	}
	if isBlockGreaterThanLatest(s.ngy, blockNum) {
		return nil, ErrRequestedBlockTooHigh
	}
	blk, err := s.ngy.BlockByNumber(ctx, blockNum)
	if err != nil {
		return nil, errors.Wrapf(err, "could not retrieve the blk information for blk number: %d", blockNum)
	}

	// Fetch validator info
	validatorInfo, err := s.ngy.GetValidatorInformation(internal_common.ParseAddr(address), blk)
	if err != nil {
		return nil, err
	}

	// Response output is the same for all versions
	return NewStructuredResponse(validatorInfo)
}

// GetValidatorSelfDelegation returns validator stake.
func (s *PublicStakingService) GetValidatorSelfDelegation(
	ctx context.Context, address string,
) (interface{}, error) {
	// Ensure node is for beacon shard
	if !isBeaconShard(s.ngy) {
		return nil, ErrNotBeaconShard
	}

	// Fetch self delegation
	selfDelegation := s.ngy.GetValidatorSelfDelegation(internal_common.ParseAddr(address)).Uint64()

	// Format the response according to the version
	switch s.version {
	case V1:
		return hexutil.Uint64(selfDelegation), nil
	case V2:
		return selfDelegation, nil
	default:
		return nil, ErrUnknownRPCVersion
	}
}

// GetValidatorTotalDelegation returns total balance stacking for validator with delegation.
func (s *PublicStakingService) GetValidatorTotalDelegation(
	ctx context.Context, address string,
) (interface{}, error) {
	// Ensure node is for beacon shard
	if s.ngy.ShardID != shard.BeaconChainShardID {
		return nil, ErrNotBeaconShard
	}

	// Fetch delegations & sum
	delegations := s.ngy.GetDelegationsByValidator(internal_common.ParseAddr(address))
	totalStake := big.NewInt(0)
	for _, delegation := range delegations {
		totalStake.Add(totalStake, delegation.Amount)
	}

	// Format the response according to the version
	switch s.version {
	case V1:
		return hexutil.Uint64(totalStake.Uint64()), nil
	case V2:
		return totalStake, nil
	default:
		return nil, ErrUnknownRPCVersion
	}
}

// GetAllDelegationInformation returns delegation information about `validatorsPageSize` validators,
// starting at `page*validatorsPageSize`.
// If page is -1, return all instead of `validatorsPageSize` elements.
// TODO(dm): optimize with single flight
func (s *PublicStakingService) GetAllDelegationInformation(
	ctx context.Context, page int,
) ([][]StructuredResponse, error) {
	if !isBeaconShard(s.ngy) {
		return nil, ErrNotBeaconShard
	}
	if page < -1 {
		return make([][]StructuredResponse, 0), nil
	}

	// Get all validators
	addresses := s.ngy.GetAllValidatorAddresses()

	// Return nothing if no delegation on page
	if page != -1 && len(addresses) <= page*validatorsPageSize {
		return make([][]StructuredResponse, 0), nil
	}

	// Set page start
	validatorsNum := len(addresses)
	start := 0
	if page != -1 {
		validatorsNum = validatorsPageSize
		start = page * validatorsPageSize
		if len(addresses)-start < validatorsPageSize {
			validatorsNum = len(addresses) - start
		}
	}

	// Fetch all delegations
	validators := make([][]StructuredResponse, validatorsNum)
	var err error
	for i := start; i < start+validatorsNum; i++ {
		validators[i-start], err = s.GetDelegationsByValidator(ctx, addresses[i].String())
		if err != nil {
			return nil, err
		}
	}

	// Response output is the same for all versions
	return validators, nil
}

// GetDelegationsByDelegator returns list of delegations for a delegator address.
func (s *PublicStakingService) GetDelegationsByDelegator(
	ctx context.Context, address string,
) ([]StructuredResponse, error) {
	if !isBeaconShard(s.ngy) {
		return nil, ErrNotBeaconShard
	}

	// Fetch delegation
	delegatorAddress := internal_common.ParseAddr(address)
	validators, delegations := s.ngy.GetDelegationsByDelegator(delegatorAddress)

	// Format response
	result := []StructuredResponse{}
	for i := range delegations {
		delegation := delegations[i]
		undelegations := make([]Undelegation, len(delegation.Undelegations))

		for j := range delegation.Undelegations {
			undelegations[j] = Undelegation{
				Amount: delegation.Undelegations[j].Amount,
				Epoch:  delegation.Undelegations[j].Epoch,
			}
		}
		valAddr, _ := internal_common.AddressToBech32(validators[i])
		delAddr, _ := internal_common.AddressToBech32(delegatorAddress)

		// Response output is the same for all versions
		del, err := NewStructuredResponse(Delegation{
			ValidatorAddress: valAddr,
			DelegatorAddress: delAddr,
			Amount:           delegation.Amount,
			Reward:           delegation.Reward,
			Undelegations:    undelegations,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, del)
	}
	return result, nil
}

// GetDelegationsByDelegatorByBlockNumber returns list of delegations for a delegator address at given block number
func (s *PublicStakingService) GetDelegationsByDelegatorByBlockNumber(
	ctx context.Context, address string, blockNumber BlockNumber,
) ([]StructuredResponse, error) {
	// Process number based on version
	blockNum := blockNumber.EthBlockNumber()

	if !isBeaconShard(s.ngy) {
		return nil, ErrNotBeaconShard
	}
	if isBlockGreaterThanLatest(s.ngy, blockNum) {
		return nil, ErrRequestedBlockTooHigh
	}

	// Fetch delegation for block number
	delegatorAddress := internal_common.ParseAddr(address)
	blk, err := s.ngy.BlockByNumber(ctx, blockNum)
	if err != nil {
		return nil, errors.Wrapf(err, "could not retrieve the blk information for blk number: %d", blockNum)
	}
	validators, delegations := s.ngy.GetDelegationsByDelegatorByBlock(delegatorAddress, blk)

	// Format response
	result := []StructuredResponse{}
	for i := range delegations {
		delegation := delegations[i]
		undelegations := make([]Undelegation, len(delegation.Undelegations))

		for j := range delegation.Undelegations {
			undelegations[j] = Undelegation{
				Amount: delegation.Undelegations[j].Amount,
				Epoch:  delegation.Undelegations[j].Epoch,
			}
		}
		valAddr, _ := internal_common.AddressToBech32(validators[i])
		delAddr, _ := internal_common.AddressToBech32(delegatorAddress)

		// Response output is the same for all versions
		del, err := NewStructuredResponse(Delegation{
			ValidatorAddress: valAddr,
			DelegatorAddress: delAddr,
			Amount:           delegation.Amount,
			Reward:           delegation.Reward,
			Undelegations:    undelegations,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, del)
	}
	return result, nil
}

// GetDelegationsByValidator returns list of delegations for a validator address.
func (s *PublicStakingService) GetDelegationsByValidator(
	ctx context.Context, address string,
) ([]StructuredResponse, error) {
	if !isBeaconShard(s.ngy) {
		return nil, ErrNotBeaconShard
	}

	// Fetch delegations
	validatorAddress := internal_common.ParseAddr(address)
	delegations := s.ngy.GetDelegationsByValidator(validatorAddress)

	// Format response
	result := []StructuredResponse{}
	for i := range delegations {
		delegation := delegations[i]
		undelegations := make([]Undelegation, len(delegation.Undelegations))

		for j := range delegation.Undelegations {
			undelegations[j] = Undelegation{
				Amount: delegation.Undelegations[j].Amount,
				Epoch:  delegation.Undelegations[j].Epoch,
			}
		}
		valAddr, _ := internal_common.AddressToBech32(validatorAddress)
		delAddr, _ := internal_common.AddressToBech32(delegation.DelegatorAddress)

		// Response output is the same for all versions
		del, err := NewStructuredResponse(Delegation{
			ValidatorAddress: valAddr,
			DelegatorAddress: delAddr,
			Amount:           delegation.Amount,
			Reward:           delegation.Reward,
			Undelegations:    undelegations,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, del)
	}
	return result, nil
}

// GetDelegationByDelegatorAndValidator returns a delegation for delegator and validator.
func (s *PublicStakingService) GetDelegationByDelegatorAndValidator(
	ctx context.Context, address string, validator string,
) (StructuredResponse, error) {
	if !isBeaconShard(s.ngy) {
		return nil, ErrNotBeaconShard
	}

	// Fetch delegations
	delegatorAddress := internal_common.ParseAddr(address)
	validatorAddress := internal_common.ParseAddr(validator)
	validators, delegations := s.ngy.GetDelegationsByDelegator(delegatorAddress)

	// Format response
	for i := range delegations {
		if validators[i] != validatorAddress {
			continue
		}
		delegation := delegations[i]
		undelegations := make([]Undelegation, len(delegation.Undelegations))

		for j := range delegation.Undelegations {
			undelegations[j] = Undelegation{
				Amount: delegation.Undelegations[j].Amount,
				Epoch:  delegation.Undelegations[j].Epoch,
			}
		}
		valAddr, _ := internal_common.AddressToBech32(validatorAddress)
		delAddr, _ := internal_common.AddressToBech32(delegatorAddress)

		// Response output is the same for all versions
		return NewStructuredResponse(Delegation{
			ValidatorAddress: valAddr,
			DelegatorAddress: delAddr,
			Amount:           delegation.Amount,
			Reward:           delegation.Reward,
			Undelegations:    undelegations,
		})
	}
	return nil, nil
}

// GetAvailableRedelegationBalance returns the amount of locked undelegated tokens
func (s *PublicStakingService) GetAvailableRedelegationBalance(
	ctx context.Context, address string,
) (*big.Int, error) {
	if !isBeaconShard(s.ngy) {
		return nil, ErrNotBeaconShard
	}

	currEpoch := s.ngy.BlockChain.CurrentHeader().Epoch()

	delegatorAddr := internal_common.ParseAddr(address)
	_, delegations := s.ngy.GetDelegationsByDelegator(delegatorAddr)

	redelegationTotal := big.NewInt(0)
	for _, d := range delegations {
		for _, u := range d.Undelegations {
			if u.Epoch.Cmp(currEpoch) < 1 { // Undelegation.Epoch < currentEpoch
				redelegationTotal.Add(redelegationTotal, u.Amount)
			}
		}
	}
	return redelegationTotal, nil
}

func isBeaconShard(ngy *ngy.nordicenergy) bool {
	return ngy.ShardID == shard.BeaconChainShardID
}
