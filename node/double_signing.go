package node

import (
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/nordicenergy/nordicenergy-core/internal/utils"
	"github.com/nordicenergy/nordicenergy-core/shard"
	"github.com/nordicenergy/nordicenergy-core/staking/slash"
)

// ProcessSlashCandidateMessage ..
func (node *Node) processSlashCandidateMessage(msgPayload []byte) {
	if node.NodeConfig.ShardID != shard.BeaconChainShardID {
		return
	}
	candidates := slash.Records{}

	if err := rlp.DecodeBytes(msgPayload, &candidates); err != nil {
		utils.Logger().Error().
			Err(err).Msg("unable to decode slash candidates message")
		return
	}

	if err := node.Blockchain().AddPendingSlashingCandidates(
		candidates,
	); err != nil {
		utils.Logger().Error().
			Err(err).Msg("unable to add slash candidates to pending ")
	}
}
