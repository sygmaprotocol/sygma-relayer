package common

import "encoding/json"

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
