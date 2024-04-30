// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package connection

import (
	"github.com/btcsuite/btcd/rpcclient"
)

type Connection struct {
	*rpcclient.Client
}

func NewBtcConnection(url string) (*Connection, error) {
	// Connect to a Bitcoin node using RPC
	connConfig := &rpcclient.ConnConfig{
		HTTPPostMode: true,
		Host:         "nd-878-662-521.p2pify.com",
		User:         "flamboyant-agnesi",
		Pass:         "sadden-demise-okay-caucus-alarm-comply",
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

/*
04404f3469c6a1cffb95ee89eb2f17f69c015133a57d08785a11a17e11dc4db4213228ceb450933ed769fe3642329cd3454883374534f685e50553d2cddde27a9682
03dca0d4d11392b7d3947d48b8d85683f4391036d5b326d291139beeeac4da201c
 04dca0d4d11392b7d3947d48b8d85683f4391036d5b326d291139beeeac4da201c

OP_DUP
OP_HASH160
7a79915f64b49046b99f1fef02551fdab32f5b0d
OP_EQUALVERIFY
OP_CHECKSIG
*/
