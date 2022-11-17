// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package client

import (
	"sync"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client"
	"github.com/centrifuge/go-substrate-rpc-client/signature"
	"github.com/centrifuge/go-substrate-rpc-client/types"
)

type SubstrateClient struct {
	*gsrpc.SubstrateAPI
	meta        types.Metadata         // Latest chain metadata
	metaLock    sync.RWMutex           // Lock metadata for updates, allows concurrent reads
	genesisHash types.Hash             // Chain genesis hash
	key         *signature.KeyringPair // Keyring used for signing
	name        string                 // Chain name
}

func NewSubstrateClient(url string, key *signature.KeyringPair, name string) (*SubstrateClient, error) {
	c := &SubstrateClient{key: key, name: name}
	api, err := gsrpc.NewSubstrateAPI(url)
	if err != nil {
		return nil, err
	}
	c.SubstrateAPI = api

	// Fetch metadata
	meta, err := c.SubstrateAPI.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, err
	}
	c.meta = *meta
	// Fetch genesis hash
	genesisHash, err := c.SubstrateAPI.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return nil, err
	}
	c.genesisHash = genesisHash
	return c, nil
}

func (c *SubstrateClient) GetMetadata() (meta types.Metadata) {
	c.metaLock.RLock()
	meta = c.meta
	c.metaLock.RUnlock()
	return meta
}

func (c *SubstrateClient) UpdateMetatdata() error {
	c.metaLock.Lock()
	meta, err := c.SubstrateAPI.RPC.State.GetMetadataLatest()
	if err != nil {
		c.metaLock.Unlock()
		return err
	}
	c.meta = *meta
	c.metaLock.Unlock()
	return nil
}
