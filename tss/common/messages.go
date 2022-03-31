package common

import "encoding/json"

type TssMessage struct {
	MsgBytes   []byte `json:"msgBytes"`
	From       string `json:"from"`
	IsBrodcast bool   `json:"isBrodcast"`
}

func MarshalTssMessage(msgBytes []byte, isBrodcast bool, from string) ([]byte, error) {
	tssMsg := &TssMessage{
		IsBrodcast: isBrodcast,
		From:       from,
		MsgBytes:   msgBytes,
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
