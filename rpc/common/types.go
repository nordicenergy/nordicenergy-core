package common

import (
	"encoding/json"

	"github.com/nordicenergy/nordicenergy-core/internal/params"
	"github.com/libp2p/go-libp2p-core/peer"
)

// BlockArgs is struct to include optional block formatting params.
type BlockArgs struct {
	WithSigners bool     `json:"withSigners"`
	InclTx      bool     `json:"inclTx"`
	FullTx      bool     `json:"fullTx"`
	Signers     []string `json:"signers"`
	InclStaking bool     `json:"inclStaking"`
}

// UnmarshalFromInterface ..
func (ba *BlockArgs) UnmarshalFromInterface(blockArgs interface{}) error {
	var args BlockArgs
	dat, err := json.Marshal(blockArgs)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(dat, &args); err != nil {
		return err
	}
	*ba = args
	return nil
}

// C ..
type C struct {
	TotalKnownPeers int `json:"total-known-peers"`
	Connected       int `json:"connected"`
	NotConnected    int `json:"not-connected"`
}

// ConsensusInternal captures consensus internal data
type ConsensusInternal struct {
	ViewID        uint64 `json:"viewId"`
	ViewChangeID  uint64 `json:"viewChangeId"`
	Mode          string `json:"mode"`
	Phase         string `json:"phase"`
	BlockNum      uint64 `json:"blocknum"`
	ConsensusTime int64  `json:"finality"`
}

// NodeMetadata captures select metadata of the RPC answering node
type NodeMetadata struct {
	BLSPublicKey   []string           `json:"blskey"`
	Version        string             `json:"version"`
	NetworkType    string             `json:"network"`
	ChainConfig    params.ChainConfig `json:"chain-config"`
	IsLeader       bool               `json:"is-leader"`
	ShardID        uint32             `json:"shard-id"`
	CurrentEpoch   uint64             `json:"current-epoch"`
	BlocksPerEpoch *uint64            `json:"blocks-per-epoch,omitempty"`
	Role           string             `json:"role"`
	DNSZnet        string             `json:"dns-znet"`
	Archival       bool               `json:"is-archival"`
	NodeBootTime   int64              `json:"node-unix-start-time"`
	PeerID         peer.ID            `json:"peerid"`
	Consensus      ConsensusInternal  `json:"consensus"`
	C              C                  `json:"p2p-connectivity"`
}

// P captures the connected peers per topic
type P struct {
	Topic string    `json:"topic"`
	Peers []peer.ID `json:"peers"`
}

// NodePeerInfo captures the peer connectivity info of the node
type NodePeerInfo struct {
	PeerID       peer.ID   `json:"peerid"`
	BlockedPeers []peer.ID `json:"blocked-peers"`
	P            []P       `json:"connected-peers"`
}