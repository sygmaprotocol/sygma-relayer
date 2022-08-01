package common

import (
	"encoding/json"

	"github.com/libp2p/go-libp2p-core/peer"
)

type TssMessage struct {
	MsgBytes    []byte `json:"msgBytes"`
	From        string `json:"from"`
	IsBroadcast bool   `json:"isBroadcast"`
}

func MarshalTssMessage(msgBytes []byte, isBroadcast bool, from string) ([]byte, error) {
	tssMsg := &TssMessage{
		IsBroadcast: isBroadcast,
		From:        from,
		MsgBytes:    msgBytes,
	}

	msgBytes, err := json.Marshal(tssMsg)
	if err != nil {
		return []byte{}, err
	}

	return msgBytes, nil
}

func UnmarshalTssMessage(msgBytes []byte) (*TssMessage, error) {
	msg := &TssMessage{}
	err := json.Unmarshal(msgBytes, msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

type StartMessage struct {
	Params []byte `json:"params"`
}

func MarshalStartMessage(params []byte) ([]byte, error) {
	startSignMessage := &StartMessage{
		Params: params,
	}

	msgBytes, err := json.Marshal(startSignMessage)
	if err != nil {
		return []byte{}, err
	}

	return msgBytes, nil
}

func UnmarshalStartMessage(msgBytes []byte) (*StartMessage, error) {
	msg := &StartMessage{}
	err := json.Unmarshal(msgBytes, msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

type FailMessage struct {
	ExcludedPeers []peer.ID `json:"excludedPeers"`
}

func MarshalFailMessage(excludedPeers []peer.ID) []byte {
	failMsg := &FailMessage{
		ExcludedPeers: excludedPeers,
	}

	msgBytes, _ := json.Marshal(failMsg)
	return msgBytes
}

func UnmarshalFailMessage(msgBytes []byte) (*FailMessage, error) {
	msg := &FailMessage{}
	err := json.Unmarshal(msgBytes, msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}
