// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package connection

import (
	"github.com/btcsuite/btcd/rpcclient"
)

type Connection struct {
	*rpcclient.Client
}

func NewBtcConnection(url string, username string, password string) (*Connection, error) {

	// Connect to a Bitcoin node using RPC
	connConfig := &rpcclient.ConnConfig{
		HTTPPostMode: true,
		Host:         url,
		User:         username,
		Pass:         password,
		DisableTLS:   false,
	}

	client, err := rpcclient.New(connConfig, nil)
	if err != nil {
		return nil, err
	}

	return &Connection{
		Client: client,
	}, nil
}
