// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package bridge

import (
	"context"
	"math/big"
	"strconv"
	"strings"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/contracts/deposit"

	"github.com/ChainSafe/chainbridge-core/types"
	"github.com/ChainSafe/sygma-relayer/chains"
	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/consts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/rs/zerolog/log"

	"github.com/sygmaprotocol/sygma-core/chains/evm/client"
	"github.com/sygmaprotocol/sygma-core/chains/evm/contracts"
	"github.com/sygmaprotocol/sygma-core/chains/evm/transactor"
	"github.com/sygmaprotocol/sygma-core/relayer/proposal"
)

const bridgeVersion = "3.1.0"

type BridgeProposal struct {
	OriginDomainID uint8
	ResourceID     [32]byte
	DepositNonce   uint64
	Data           []byte
}

type ChainClient interface {
	client.Client
	ChainID(ctx context.Context) (*big.Int, error)
}

type BridgeContract struct {
	contracts.Contract
	client ChainClient
}

func NewBridgeContract(
	client ChainClient,
	bridgeContractAddress common.Address,
	transactor transactor.Transactor,
) *BridgeContract {
	a, _ := abi.JSON(strings.NewReader(consts.BridgeABI))
	return &BridgeContract{
		Contract: contracts.NewContract(bridgeContractAddress, a, nil, client, transactor),
		client:   client,
	}
}

func (c *BridgeContract) deposit(
	resourceID types.ResourceID,
	destDomainID uint8,
	data []byte,
	feeData []byte,
	opts transactor.TransactOptions,
) (*common.Hash, error) {
	return c.ExecuteTransaction(
		"deposit",
		opts,
		destDomainID, resourceID, data, feeData,
	)
}

func (c *BridgeContract) Erc20Deposit(
	recipient []byte,
	amount *big.Int,
	resourceID types.ResourceID,
	destDomainID uint8,
	feeData []byte,
	opts transactor.TransactOptions,
) (*common.Hash, error) {
	log.Debug().
		Str("recipient", hexutil.Encode(recipient)).
		Str("resourceID", hexutil.Encode(resourceID[:])).
		Str("amount", amount.String()).
		Uint8("destDomainID", destDomainID).
		Hex("feeData", feeData).
		Msgf("ERC20 deposit")
	var data []byte
	if opts.Priority == 0 {
		data = deposit.ConstructErc20DepositData(recipient, amount)
	} else {
		data = deposit.ConstructErc20DepositDataWithPriority(recipient, amount, opts.Priority)
	}

	txHash, err := c.deposit(resourceID, destDomainID, data, feeData, opts)
	if err != nil {
		log.Error().Err(err)
		return nil, err
	}
	return txHash, err
}

func (c *BridgeContract) Erc721Deposit(
	tokenId *big.Int,
	metadata string,
	recipient common.Address,
	resourceID types.ResourceID,
	destDomainID uint8,
	feeData []byte,
	opts transactor.TransactOptions,
) (*common.Hash, error) {
	log.Debug().
		Str("recipient", recipient.String()).
		Str("resourceID", hexutil.Encode(resourceID[:])).
		Str("tokenID", tokenId.String()).
		Uint8("destDomainID", destDomainID).
		Hex("feeData", feeData).
		Msgf("ERC721 deposit")
	var data []byte
	if opts.Priority == 0 {
		data = deposit.ConstructErc721DepositData(recipient.Bytes(), tokenId, []byte(metadata))
	} else {
		data = deposit.ConstructErc721DepositDataWithPriority(recipient.Bytes(), tokenId, []byte(metadata), opts.Priority)
	}

	txHash, err := c.deposit(resourceID, destDomainID, data, feeData, opts)
	if err != nil {
		log.Error().Err(err)
		return nil, err
	}
	return txHash, err
}

func (c *BridgeContract) GenericDeposit(
	metadata []byte,
	resourceID types.ResourceID,
	destDomainID uint8,
	feeData []byte,
	opts transactor.TransactOptions,
) (*common.Hash, error) {
	log.Debug().
		Str("resourceID", hexutil.Encode(resourceID[:])).
		Uint8("destDomainID", destDomainID).
		Hex("feeData", feeData).
		Msgf("Generic deposit")
	data := deposit.ConstructGenericDepositData(metadata)

	txHash, err := c.deposit(resourceID, destDomainID, data, feeData, opts)
	if err != nil {
		log.Error().Err(err)
		return nil, err
	}
	return txHash, err
}

func (c *BridgeContract) PermissionlessGenericDeposit(
	metadata []byte,
	executeFunctionSig string,
	executeContractAddress *common.Address,
	depositor *common.Address,
	maxFee *big.Int,
	resourceID types.ResourceID,
	destDomainID uint8,
	feeData []byte,
	opts transactor.TransactOptions,
) (*common.Hash, error) {
	log.Debug().
		Str("resourceID", hexutil.Encode(resourceID[:])).
		Uint8("destDomainID", destDomainID).
		Hex("feeData", feeData).
		Msgf("Permissionless Generic deposit")
	data := ConstructPermissionlessGenericDepositData(metadata, []byte(executeFunctionSig), executeContractAddress.Bytes(), depositor.Bytes(), maxFee)
	txHash, err := c.deposit(
		resourceID,
		destDomainID,
		data,
		feeData,
		opts,
	)
	if err != nil {
		log.Error().Err(err)
		return nil, err
	}
	return txHash, err
}

func (c *BridgeContract) ExecuteProposal(
	proposal *chains.Proposal,
	signature []byte,
	opts transactor.TransactOptions,
) (*common.Hash, error) {
	log.Debug().
		Str("depositNonce", strconv.FormatUint(proposal.DepositNonce, 10)).
		Str("resourceID", hexutil.Encode(proposal.ResourceID[:])).
		Msgf("Execute proposal")
	return c.ExecuteTransaction(
		"executeProposal",
		opts,
		proposal.OriginDomainID, proposal.DepositNonce, proposal.Data, proposal.ResourceID, signature,
	)
}

func (c *BridgeContract) ExecuteProposals(
	proposals []*proposal.Proposal,
	signature []byte,
	opts transactor.TransactOptions,
) (*common.Hash, error) {
	bridgeProposals := make([]proposal.Proposal, 0)
	for _, prop := range proposals {
		bridgeProposals = append(bridgeProposals, proposal.Proposal{
			Source: prop.Source,
			Data: chains.TransferProposalData{
				DepositNonce: prop.Data.(chains.TransferProposalData).DepositNonce,
				ResourceId:   prop.Data.(chains.TransferProposalData).ResourceId,
				Data:         prop.Data.(chains.TransferProposalData).Data,
			},
		})
	}

	return c.ExecuteTransaction(
		"executeProposals",
		opts,
		bridgeProposals,
		signature,
	)
}

func (c *BridgeContract) ProposalsHash(proposals []*proposal.Proposal) ([]byte, error) {
	chainID, err := c.client.ChainID(context.Background())
	if err != nil {
		return []byte{}, err
	}
	return chains.ProposalsHash(proposals, chainID.Int64(), c.ContractAddress().Hex(), bridgeVersion)
}

func (c *BridgeContract) IsProposalExecuted(p *proposal.Proposal) (bool, error) {
	transferProposal := &chains.TransferProposal{
		Source:      p.Source,
		Destination: p.Destination,
		Data: chains.TransferProposalData{
			DepositNonce: p.Data.(chains.TransferProposalData).DepositNonce,
			ResourceId:   p.Data.(chains.TransferProposalData).ResourceId,
			Metadata:     p.Data.(chains.TransferProposalData).Metadata,
			Data:         p.Data.(chains.TransferProposalData).Data,
		},
		Type: p.Type,
	}
	log.Debug().
		Str("depositNonce", strconv.FormatUint(transferProposal.Data.DepositNonce, 10)).
		Str("resourceID", hexutil.Encode(transferProposal.Data.ResourceId[:])).
		Msg("Getting is proposal executed")
	res, err := c.CallContract("isProposalExecuted", p.Source, big.NewInt(int64(p.Data.(chains.TransferProposalData).DepositNonce)))
	if err != nil {
		return false, err
	}
	out := *abi.ConvertType(res[0], new(bool)).(*bool)
	return out, nil
}

func (c *BridgeContract) GetHandlerAddressForResourceID(
	resourceID [32]byte,
) (common.Address, error) {
	log.Debug().Msgf("Getting handler address for resource %s", hexutil.Encode(resourceID[:]))
	res, err := c.CallContract("_resourceIDToHandlerAddress", resourceID)
	if err != nil {
		return common.Address{}, err
	}
	out := *abi.ConvertType(res[0], new(common.Address)).(*common.Address)
	return out, nil
}

func (c *BridgeContract) Retry(hash common.Hash, opts transactor.TransactOptions) (*common.Hash, error) {
	log.Debug().Msgf("Retrying deposit from transaction: %s", hash.Hex())
	return c.ExecuteTransaction("retry", opts, hash.Hex())
}
