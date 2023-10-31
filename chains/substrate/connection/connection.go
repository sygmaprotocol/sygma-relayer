// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package connection

import (
	"sync"

	"github.com/centrifuge/go-substrate-rpc-client/v4/client"
	"github.com/centrifuge/go-substrate-rpc-client/v4/registry/parser"
	"github.com/centrifuge/go-substrate-rpc-client/v4/registry/retriever"
	"github.com/centrifuge/go-substrate-rpc-client/v4/registry/state"

	"github.com/centrifuge/go-substrate-rpc-client/v4/rpc"
	"github.com/centrifuge/go-substrate-rpc-client/v4/rpc/chain"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type Connection struct {
	chain.Chain
	client.Client
	*rpc.RPC
	meta        types.Metadata // Latest chain metadata
	metaLock    sync.RWMutex   // Lock metadata for updates, allows concurrent reads
	GenesisHash types.Hash     // Chain genesis hash
}

func NewSubstrateConnection(url string) (*Connection, error) {
	client, err := client.Connect(url)
	if err != nil {
		return nil, err
	}
	rpc, err := rpc.NewRPC(client)
	if err != nil {
		return nil, err
	}

	meta, err := rpc.State.GetMetadataLatest()
	if err != nil {
		return nil, err
	}
	genesisHash, err := rpc.Chain.GetBlockHash(0)
	if err != nil {
		return nil, err
	}

	return &Connection{
		meta: *meta,

		RPC:         rpc,
		Chain:       rpc.Chain,
		Client:      client,
		GenesisHash: genesisHash,
	}, nil
}

func (c *Connection) GetMetadata() (meta types.Metadata) {
	c.metaLock.RLock()
	meta = c.meta
	c.metaLock.RUnlock()
	return meta
}

func (c *Connection) UpdateMetatdata() error {
	c.metaLock.Lock()
	meta, err := c.RPC.State.GetMetadataLatest()
	if err != nil {
		c.metaLock.Unlock()
		return err
	}
	c.meta = *meta
	c.metaLock.Unlock()
	return nil
}

func (c *Connection) GetBlockEvents(hash types.Hash) ([]*parser.Event, error) {
	provider := state.NewEventProvider(c.State)
	eventRetriever, err := retriever.NewDefaultEventRetriever(provider, c.State)
	if err != nil {
		return nil, err
	}

	evts, err := eventRetriever.GetEvents(hash)
	if err != nil {
		return nil, err
	}
	return evts, nil
}
