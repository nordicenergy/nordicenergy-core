package rpc

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/nordicenergy/nordicenergy-core/ngy"
)

// PublicnordicenergyService provides an API to access nordicenergy related information.
// It offers only methods that operate on public data that is freely available to anynet.
type PublicnordicenergyService struct {
	ngy     *ngy.nordicenergy
	version Version
}

// NewPublicnordicenergyAPI creates a new API for the RPC interface
func NewPublicnordicenergyAPI(ngy *ngy.nordicenergy, version Version) rpc.API {
	return rpc.API{
		Namespace: version.Namespace(),
		Version:   APIVersion,
		Service:   &PublicnordicenergyService{ngy, version},
		Public:    true,
	}
}

// ProtocolVersion returns the current nordicenergy protocol version this node supports
// Note that the return type is an interface to account for the different versions
func (s *PublicnordicenergyService) ProtocolVersion(
	ctx context.Context,
) (interface{}, error) {
	// Format response according to version
	switch s.version {
	case V1, Eth:
		return hexutil.Uint(s.ngy.ProtocolVersion()), nil
	case V2:
		return s.ngy.ProtocolVersion(), nil
	default:
		return nil, ErrUnknownRPCVersion
	}
}

// Syncing returns false in case the node is currently not syncing with the network. It can be up to date or has not
// yet received the latest block headers from its pears. In case it is synchronizing:
// - startingBlock: block number this node started to synchronise from
// - currentBlock:  block number this node is currently importing
// - highestBlock:  block number of the highest block header this node has received from peers
// - pulledStates:  number of state entries processed until now
// - knownStates:   number of known state entries that still need to be pulled
func (s *PublicnordicenergyService) Syncing(
	ctx context.Context,
) (interface{}, error) {
	// TODO(dm): find our Downloader module for syncing blocks
	return false, nil
}

// GasPrice returns a suggestion for a gas price.
// Note that the return type is an interface to account for the different versions
func (s *PublicnordicenergyService) GasPrice(ctx context.Context) (interface{}, error) {
	// TODO(dm): add SuggestPrice API
	// Format response according to version
	switch s.version {
	case V1, Eth:
		return (*hexutil.Big)(big.NewInt(1)), nil
	case V2:
		return 1, nil
	default:
		return nil, ErrUnknownRPCVersion
	}
}

// GetNodeMetadata produces a NodeMetadata record, data is from the answering RPC node
func (s *PublicnordicenergyService) GetNodeMetadata(
	ctx context.Context,
) (StructuredResponse, error) {
	// Response output is the same for all versions
	return NewStructuredResponse(s.ngy.GetNodeMetadata())
}

// GetPeerInfo produces a NodePeerInfo record
func (s *PublicnordicenergyService) GetPeerInfo(
	ctx context.Context,
) (StructuredResponse, error) {
	// Response output is the same for all versions
	return NewStructuredResponse(s.ngy.GetPeerInfo())
}
