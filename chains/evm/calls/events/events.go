// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package events

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type EventSig string

func (es EventSig) GetTopic() common.Hash {
	return crypto.Keccak256Hash([]byte(es))
}

const (
	DepositSig           EventSig = "Deposit(uint8,bytes32,uint64,address,bytes,bytes)"
	StartKeygenSig       EventSig = "StartKeygen()"
	StartFrostKeygenSig  EventSig = "StartedFROSTKeygen()"
	KeyRefreshSig        EventSig = "KeyRefresh(string)"
	ProposalExecutionSig EventSig = "ProposalExecution(uint8,uint64,bytes32,bytes)"
	FeeChangedSig        EventSig = "FeeChanged(uint256)"
	RetrySig             EventSig = "Retry(string)"
	FeeHandlerChanged    EventSig = "FeeHandlerChanged(address)"
)

// Refresh struct holds key refresh event data
type Refresh struct {
	// SHA1 hash of topology file
	Hash string
}

type RetryEvent struct {
	TxHash string
}

type Deposit struct {
	// ID of chain deposit will be bridged to
	DestinationDomainID uint8
	// ResourceID used to find address of handler to be used for deposit
	ResourceID [32]byte
	// Nonce of deposit
	DepositNonce uint64
	// Address of sender (msg.sender: user)
	SenderAddress common.Address
	// Additional data to be passed to specified handler
	Data []byte
	// ERC20Handler: responds with empty data
	// ERC721Handler: responds with deposited token metadata acquired by calling a tokenURI method in the token contract
	// GenericHandler: responds with the raw bytes returned from the call to the target contract
	HandlerResponse []byte
}
