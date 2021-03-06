package node

import (
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/nordicenergy/nordicenergy-core/core/types"
	"github.com/nordicenergy/nordicenergy-core/ngy"
	"github.com/nordicenergy/nordicenergy-core/rosetta"
	ngy_rpc "github.com/nordicenergy/nordicenergy-core/rpc"
	rpc_common "github.com/nordicenergy/nordicenergy-core/rpc/common"
	"github.com/nordicenergy/nordicenergy-core/rpc/filters"
	"github.com/libp2p/go-libp2p-core/peer"
)

// IsCurrentlyLeader exposes if node is currently the leader node
func (node *Node) IsCurrentlyLeader() bool {
	return node.Consensus.IsLeader()
}

// PeerConnectivity ..
func (node *Node) PeerConnectivity() (int, int, int) {
	return node.host.C()
}

// ListPeer return list of peers for a certain topic
func (node *Node) ListPeer(topic string) []peer.ID {
	return node.host.ListPeer(topic)
}

// ListTopic return list of topics the node subscribed
func (node *Node) ListTopic() []string {
	return node.host.ListTopic()
}

// ListBlockedPeer return list of blocked peers
func (node *Node) ListBlockedPeer() []peer.ID {
	return node.host.ListBlockedPeer()
}

// PendingCXReceipts returns node.pendingCXReceiptsProof
func (node *Node) PendingCXReceipts() []*types.CXReceiptsProof {
	cxReceipts := make([]*types.CXReceiptsProof, len(node.pendingCXReceipts))
	i := 0
	for _, cxReceipt := range node.pendingCXReceipts {
		cxReceipts[i] = cxReceipt
		i++
	}
	return cxReceipts
}

// ReportStakingErrorSink is the report of failed staking transactions this node has (held in memory only)
func (node *Node) ReportStakingErrorSink() types.TransactinetrrorReports {
	return node.TransactinetrrorSink.StakingReport()
}

// GetNodeBootTime ..
func (node *Node) GetNodeBootTime() int64 {
	return node.unixTimeAtNodeStart
}

// ReportPlainErrorSink is the report of failed transactions this node has (held in memory only)
func (node *Node) ReportPlainErrorSink() types.TransactinetrrorReports {
	return node.TransactinetrrorSink.PlainReport()
}

// StartRPC start RPC service
func (node *Node) StartRPC() error {
	nordicenergy := ngy.New(node, node.TxPool, node.CxPool, node.Consensus.ShardID)

	// Gather all the possible APIs to surface
	apis := node.APIs(nordicenergy)

	for _, service := range node.serviceManager.GetServices() {
		apis = append(apis, service.APIs()...)
	}

	return ngy_rpc.StartServers(nordicenergy, apis, node.NodeConfig.RPCServer)
}

// StopRPC stop RPC service
func (node *Node) StopRPC() error {
	return ngy_rpc.StopServers()
}

// StartRosetta start rosetta service
func (node *Node) StartRosetta() error {
	nordicenergy := ngy.New(node, node.TxPool, node.CxPool, node.Consensus.ShardID)
	return rosetta.StartServers(nordicenergy, node.NodeConfig.RosettaServer)
}

// StopRosetta stops rosetta service
func (node *Node) StopRosetta() error {
	return rosetta.StopServers()
}

// APIs return the collection of local RPC services.
// NOTE, some of these services probably need to be moved to somewhere else.
func (node *Node) APIs(nordicenergy *ngy.nordicenergy) []rpc.API {
	// Append all the local APIs and return
	return []rpc.API{
		ngy_rpc.NewPublicNetAPI(node.host, nordicenergy.ChainID, ngy_rpc.V1),
		ngy_rpc.NewPublicNetAPI(node.host, nordicenergy.ChainID, ngy_rpc.V2),
		ngy_rpc.NewPublicNetAPI(node.host, nordicenergy.ChainID, ngy_rpc.Eth),
		ngy_rpc.NewPublicWeb3API(),
		filters.NewPublicFilterAPI(nordicenergy, false, "ngy"),
		filters.NewPublicFilterAPI(nordicenergy, false, "eth"),
	}
}

// GetConsensusMode returns the current consensus mode
func (node *Node) GetConsensusMode() string {
	return node.Consensus.GetConsensusMode()
}

// GetConsensusPhase returns the current consensus phase
func (node *Node) GetConsensusPhase() string {
	return node.Consensus.GetConsensusPhase()
}

// GetConsensusViewChangingID returns the view changing ID
func (node *Node) GetConsensusViewChangingID() uint64 {
	return node.Consensus.GetViewChangingID()
}

// GetConsensusCurViewID returns the current view ID
func (node *Node) GetConsensusCurViewID() uint64 {
	return node.Consensus.GetCurBlockViewID()
}

// GetConsensusBlockNum returns the current block number of the consensus
func (node *Node) GetConsensusBlockNum() uint64 {
	return node.Consensus.GetBlockNum()
}

// GetConsensusInternal returns consensus internal data
func (node *Node) GetConsensusInternal() rpc_common.ConsensusInternal {
	return rpc_common.ConsensusInternal{
		ViewID:        node.GetConsensusCurViewID(),
		ViewChangeID:  node.GetConsensusViewChangingID(),
		Mode:          node.GetConsensusMode(),
		Phase:         node.GetConsensusPhase(),
		BlockNum:      node.GetConsensusBlockNum(),
		ConsensusTime: node.Consensus.GetFinality(),
	}
}
