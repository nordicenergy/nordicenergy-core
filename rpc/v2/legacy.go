package v2

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/nordicenergy/nordicenergy-core/ngy"
	internal_common "github.com/nordicenergy/nordicenergy-core/internal/common"
)

// PublicLegacyService provides an API to access the nordicenergy blockchain.
// Services here are legacy methods, specific to the V1 RPC that can be deprecated in the future.
type PublicLegacyService struct {
	ngy *ngy.nordicenergy
}

// NewPublicLegacyAPI creates a new API for the RPC interface
func NewPublicLegacyAPI(ngy *ngy.nordicenergy, namespace string) rpc.API {
	if namespace == "" {
		namespace = "ngyv2"
	}

	return rpc.API{
		Namespace: namespace,
		Version:   "1.0",
		Service:   &PublicLegacyService{ngy},
		Public:    true,
	}
}

// GetBalance returns the amount of Atto for the given address in the state of the
// given block number.
func (s *PublicLegacyService) GetBalance(
	ctx context.Context, address string,
) (*big.Int, error) {
	addr := internal_common.ParseAddr(address)
	balance, err := s.ngy.GetBalance(ctx, addr, rpc.BlockNumber(-1))
	if err != nil {
		return nil, err
	}
	return balance, nil
}
