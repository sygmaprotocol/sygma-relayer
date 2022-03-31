package libp2p

import "github.com/ChainSafe/chainbridge-core/mpc/communication"

const (
	// TssKeyGenMsg message type used for communicating key generation.
	TssKeyGenMsg communication.ChainBridgeMessageType = iota
	// TssKeySignMsg message type used for communicating signature for specific message.
	TssKeySignMsg
	// TssInitiateMsg message type sent by the leader to signify preparation for a tss process.
	TssInitiateMsg
	// TssStartMsg message type sent by a leader to signify the start of a tss process after parties sent the ready message.
	TssStartMsg
	// TssReadyMsg message type sent by parties after the leader sends TssInitiateMsg to signify they are ready for the tss process.
	TssReadyMsg
	// TssReshareMsg message type used for resharing tss messages.
	TssReshareMsg
	// CoordinatorElectionMsg message type used to communicate that new election process needs to start.
	CoordinatorElectionMsg
	// CoordinatorAliveMsg  message type used to respond on CoordinatorElectionMsg message, signaling that peer is alive and ready for new election process.
	CoordinatorAliveMsg
	// CoordinatorEndMsg message type used to communicate that peer is going offline and will not participate in the future.
	CoordinatorEndMsg
	// CoordinatorSelectMsg message type used to communicate that sender has pronounced itself as a leader.
	CoordinatorSelectMsg
	// CoordinatorPingMsg message type used to check if the current coordinator is alive.
	CoordinatorPingMsg
	// CoordinatorPingResponseMsg message type used to respond on CoordinatorPingMsg message.
	CoordinatorPingResponseMsg
)

// StrMsg converts ChainBridgeMessageType to string
func StrMsg(msgType communication.ChainBridgeMessageType) string {
	switch msgType {
	case TssKeyGenMsg:
		return "TssKeyGenMsg"
	case TssKeySignMsg:
		return "TssKeySignMsg"
	case TssInitiateMsg:
		return "TssInitiateMsg"
	case TssStartMsg:
		return "TssStartMsg"
	case TssReadyMsg:
		return "TssReadyMsg"
	case TssReshareMsg:
		return "TssReshareMsg"
	case CoordinatorElectionMsg:
		return "CoordinatorElectionMsg"
	case CoordinatorAliveMsg:
		return "CoordinatorAliveMsg"
	case CoordinatorEndMsg:
		return "CoordinatorEndMsg"
	case CoordinatorSelectMsg:
		return "CoordinatorSelectMsg"
	case CoordinatorPingMsg:
		return "CoordinatorPingMsg"
	case CoordinatorPingResponseMsg:
		return "CoordinatorPingResponseMsg"
	default:
		return "UnknownMsg"
	}
}
