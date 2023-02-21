// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package client

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"

	"github.com/ChainSafe/sygma-relayer/chains/substrate/connection"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/rs/zerolog/log"
)

type SubstrateClient struct {
	key       *signature.KeyringPair // Keyring used for signing
	nonceLock sync.Mutex             // Locks nonce for updates
	nonce     types.U32              // Latest account nonce

	Conn    *connection.Connection
	ChainID *big.Int
}

func NewSubstrateClient(conn *connection.Connection, key *signature.KeyringPair, chainID *big.Int) *SubstrateClient {
	return &SubstrateClient{
		key:     key,
		Conn:    conn,
		ChainID: chainID,
	}
}

// Transact constructs and submits an extrinsic to call the method with the given arguments.
// All args are passed directly into GSRPC. GSRPC types are recommended to avoid serialization inconsistencies.
func (c *SubstrateClient) Transact(method string, args ...interface{}) (*types.Hash, error) {
	log.Debug().Msgf("Submitting substrate call... method %s, sender %s", method, c.key.Address)

	// Create call and extrinsic
	meta := c.Conn.GetMetadata()
	call, err := types.NewCall(
		&meta,
		method,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to construct call: %w", err)
	}

	ext := types.NewExtrinsic(call)
	// Get latest runtime version
	rv, err := c.Conn.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return nil, err
	}

	c.nonceLock.Lock()
	defer c.nonceLock.Unlock()

	nonce, err := c.nextNonce(&meta)
	if err != nil {
		return nil, err
	}

	// Sign the extrinsic
	o := types.SignatureOptions{
		BlockHash:          c.Conn.GenesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        c.Conn.GenesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}
	h, err := c.signAndSendTransaction(o, ext)
	if err != nil {
		return nil, fmt.Errorf("submission of extrinsic failed: %w", err)
	}

	log.Debug().Msgf("Extinsic call succededed... method %s, sender %s, nonce %d", method, c.key.Address, nonce)
	c.nonce = nonce + 1

	return &h, nil
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

func (c *SubstrateClient) signAndSendTransaction(opts types.SignatureOptions, ext types.Extrinsic) (types.Hash, error) {
	err := ext.Sign(*c.key, opts)
	if err != nil {
		return types.Hash{}, err
	}

	hash, err := c.sendRawTransaction(ext)
	if err != nil {
		return types.Hash{}, err
	}
	return hash, nil
}

// SendRawTransaction accepts rlp-encode of signed transaction and sends it via RPC call
func (c *SubstrateClient) sendRawTransaction(ext types.Extrinsic) (types.Hash, error) {
	enc, err := codec.EncodeToHex(ext)
	if err != nil {
		return types.Hash{}, err
	}

	var res string
	err = c.Conn.Call(&res, "author_submitExtrinsic", enc)
	if err != nil {
		return types.Hash{}, err
	}

	return types.NewHashFromHexString(res)
}
