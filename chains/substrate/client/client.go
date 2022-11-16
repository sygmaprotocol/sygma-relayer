// Copyright 2020 ChainSafe Systems
// SPDX-License-Identifier: LGPL-3.0-only

package client

import (
	"sync"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client"
	"github.com/centrifuge/go-substrate-rpc-client/signature"
	"github.com/centrifuge/go-substrate-rpc-client/types"
)

type Connection struct {
	Api         *gsrpc.SubstrateAPIu
	url         string                 // API endpoint
	name        string                 // Chain name
	meta        types.Metadata         // Latest chain metadata
	metaLock    sync.RWMutex           // Lock metadata for updates, allows concurrent reads
	genesisHash types.Hash             // Chain genesis hash
	key         *signature.KeyringPair // Keyring used for signing
}

func NewConnection(url string, name string, key *signature.KeyringPair) *Connection {
	return &Connection{url: url, name: name, key: key}
}

func (c *Connection) GetMetadata() (meta types.Metadata) {
	c.metaLock.RLock()
	meta = c.meta
	c.metaLock.RUnlock()
	return meta
}

func (c *Connection) UpdateMetatdata() error {
	c.metaLock.Lock()
	meta, err := c.Api.RPC.State.GetMetadataLatest()
	if err != nil {
		c.metaLock.Unlock()
		return err
	}
	c.meta = *meta
	c.metaLock.Unlock()
	return nil
}

func (c *Connection) Connect() error {
	api, err := gsrpc.NewSubstrateAPI(c.url)
	if err != nil {
		return err
	}
	c.Api = api

	// Fetch metadata
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return err
	}
	c.meta = *meta

	// Fetch genesis hash
	genesisHash, err := c.Api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return err
	}
	c.genesisHash = genesisHash
	return nil
}
