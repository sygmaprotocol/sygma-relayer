// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package evm

import (
	"github.com/ChainSafe/chainbridge-core/chains/evm"
	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/rs/zerolog/log"
)

type BatchProposalExecutor interface {
	Execute(msgs []*message.Message) error
}

type EVMChain struct {
	evm.EVMChain
	executor BatchProposalExecutor
}

func NewEVMChain(evmChain evm.EVMChain, executor BatchProposalExecutor) *EVMChain {
	return &EVMChain{
		EVMChain: evmChain,
		executor: executor,
	}
}

func (c *EVMChain) Write(msgs []*message.Message) {
	err := c.executor.Execute(msgs)
	if err != nil {
		log.Err(err).Msgf("error writing messages %+v", msgs)
	}
}
