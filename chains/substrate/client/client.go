// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package client

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/rpc/author"

	"github.com/ChainSafe/sygma-relayer/chains/substrate"
	"github.com/ChainSafe/sygma-relayer/chains/substrate/connection"
	"github.com/ChainSafe/sygma-relayer/chains/substrate/events"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/rs/zerolog/log"
)

type SubstrateClient struct {
	key       *signature.KeyringPair // Keyring used for signing
	nonceLock sync.Mutex             // Locks nonce for updates
	nonce     types.U32              // Latest account nonce
	tip       uint64
	Conn      *connection.Connection
	ChainID   *big.Int
}

func NewSubstrateClient(conn *connection.Connection, key *signature.KeyringPair, chainID *big.Int, tip uint64) *SubstrateClient {
	return &SubstrateClient{
		key:     key,
		Conn:    conn,
		ChainID: chainID,
		tip:     tip,
	}
}

// Transact constructs and submits an extrinsic to call the method with the given arguments.
// All args are passed directly into GSRPC. GSRPC types are recommended to avoid serialization inconsistencies.
func (c *SubstrateClient) Transact(method string, args ...interface{}) (string, *author.ExtrinsicStatusSubscription, error) {
	log.Debug().Msgf("Submitting substrate call... method %s, sender %s", method, c.key.Address)

	// Create call and extrinsic
	meta := c.Conn.GetMetadata()
	call, err := types.NewCall(
		&meta,
		method,
		args...,
	)
	if err != nil {
		return "", nil, fmt.Errorf("failed to construct call: %w", err)
	}

	ext := types.NewExtrinsic(call)
	// Get latest runtime version
	rv, err := c.Conn.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return "", nil, err
	}

	c.nonceLock.Lock()
	defer c.nonceLock.Unlock()

	nonce, err := c.nextNonce(&meta)
	if err != nil {
		return "", nil, err
	}

	// Sign the extrinsic
	o := types.SignatureOptions{
		BlockHash:          c.Conn.GenesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        c.Conn.GenesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(c.tip),
		TransactionVersion: rv.TransactionVersion,
	}
	sub, err := c.submitAndWatchExtrinsic(o, &ext)
	if err != nil {
		return "", nil, fmt.Errorf("submission of extrinsic failed: %w", err)
	}

	hash, err := substrate.ExtrinsicHash(ext)
	if err != nil {
		return "", nil, err
	}

	log.Info().Str("extrinsic", hash).Msgf("Extrinsic call submitted... method %s, sender %s, nonce %d", method, c.key.Address, nonce)
	c.nonce = nonce + 1

	return hash, sub, nil
}

func (c *SubstrateClient) TrackExtrinsic(extHash string, sub *author.ExtrinsicStatusSubscription, errChn chan error) {
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(time.Minute*10))
	defer sub.Unsubscribe()
	subChan := sub.Chan()
	for {
		select {
		case status := <-subChan:
			{
				if status.IsInBlock {
					log.Debug().Str("extrinsic", extHash).Msgf("Extrinsic in block with hash: %#x", status.AsInBlock)
				}
				if status.IsFinalized {
					log.Info().Str("extrinsic", extHash).Msgf("Extrinsic is finalized in block with hash: %#x", status.AsFinalized)
					err := c.checkExtrinsicSuccess(extHash, status.AsFinalized)
					errChn <- err
					return
				}
			}
		case <-ctx.Done():
			errChn <- fmt.Errorf("extrinsic has timed out")
			return
		}
	}
}

func (c *SubstrateClient) nextNonce(meta *types.Metadata) (types.U32, error) {
	key, err := types.CreateStorageKey(meta, "System", "Account", c.key.PublicKey, nil)
	if err != nil {
		return 0, err
	}

	var latestNonce types.U32
	var acct types.AccountInfo
	exists, err := c.Conn.RPC.State.GetStorageLatest(key, &acct)
	if err != nil {
		return 0, err
	}

	if !exists {
		latestNonce = 0
	} else {
		latestNonce = acct.Nonce
	}

	if latestNonce < c.nonce {
		return c.nonce, nil
	}

	return latestNonce, nil
}

func (c *SubstrateClient) submitAndWatchExtrinsic(opts types.SignatureOptions, ext *types.Extrinsic) (*author.ExtrinsicStatusSubscription, error) {
	err := ext.Sign(*c.key, opts)
	if err != nil {
		return nil, err
	}

	sub, err := c.Conn.RPC.Author.SubmitAndWatchExtrinsic(*ext)
	if err != nil {
		return nil, err
	}

	return sub, nil
}

func (c *SubstrateClient) checkExtrinsicSuccess(extHash string, blockHash types.Hash) error {
	block, err := c.Conn.Chain.GetBlock(blockHash)
	if err != nil {
		return err
	}

	evts, err := c.Conn.GetBlockEvents(blockHash)
	if err != nil {
		return err
	}

	for _, event := range evts {
		index := event.Phase.AsApplyExtrinsic
		hash, err := substrate.ExtrinsicHash(block.Block.Extrinsics[index])
		if err != nil {
			return err
		}

		if extHash != hash {
			continue
		}

		if event.Name == events.ExtrinsicFailedEvent {
			return fmt.Errorf("extrinsic failed")
		}
		if event.Name == events.ExtrinsicSuccessEvent {
			return nil
		}
	}

	return fmt.Errorf("no event found")
}
