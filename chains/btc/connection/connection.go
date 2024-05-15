// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package connection

import (
	"fmt"

	"github.com/btcsuite/btcd/rpcclient"
)

type Connection struct {
	*rpcclient.Client
}

func NewBtcConnection(url string) (*Connection, error) {
	// Connect to a Bitcoin node using RPC
	connConfig := &rpcclient.ConnConfig{
		HTTPPostMode: true,
		Host:         url,
		User:         "user",
		Pass:         "password",
		DisableTLS:   true,
	}

	client, err := rpcclient.New(connConfig, nil)
	if err != nil {
		return nil, err
	}

	info, err := client.GetBlockChainInfo()
	if err != nil {
		return nil, err
	}
	fmt.Println(info)

	return &Connection{
		Client: client,
	}, nil
}
