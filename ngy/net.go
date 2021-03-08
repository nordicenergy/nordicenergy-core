package ngy

import (
	nodeconfig "github.com/nordicenergy/nordicenergy-core/internal/configs/node"
	commonRPC "github.com/nordicenergy/nordicenergy-core/rpc/common"
	"github.com/nordicenergy/nordicenergy-core/staking/network"
	"github.com/libp2p/go-libp2p-core/peer"
)

// GetCurrentUtilityMetrics ..
func (ngy *nordicenergy) GetCurrentUtilityMetrics() (*network.UtilityMetric, error) {
	return network.NewUtilityMetricSnapshot(ngy.BlockChain)
}

// GetPeerInfo returns the peer info to the node, including blocked peer, connected peer, number of peers
func (ngy *nordicenergy) GetPeerInfo() commonRPC.NodePeerInfo {

	topics := ngy.NodeAPI.ListTopic()
	p := make([]commonRPC.P, len(topics))

	for i, t := range topics {
		topicPeer := ngy.NodeAPI.ListPeer(t)
		p[i].Topic = t
		p[i].Peers = make([]peer.ID, len(topicPeer))
		copy(p[i].Peers, topicPeer)
	}

	return commonRPC.NodePeerInfo{
		PeerID:       nodeconfig.GetPeerID(),
		BlockedPeers: ngy.NodeAPI.ListBlockedPeer(),
		P:            p,
	}
}
