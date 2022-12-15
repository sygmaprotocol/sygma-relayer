// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package bridge

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/contracts"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/contracts/deposit"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor"
	"github.com/ChainSafe/chainbridge-core/chains/evm/executor/proposal"
	"github.com/ChainSafe/chainbridge-core/types"
	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/consts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/rs/zerolog/log"
)

type BridgeProposal struct {
	OriginDomainID uint8
	ResourceID     [32]byte
	DepositNonce   uint64
	Data           []byte
}

type ChainClient interface {
	calls.ContractCallerDispatcher
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
	recipient common.Address,
	amount *big.Int,
	resourceID types.ResourceID,
	destDomainID uint8,
	feeData []byte,
	opts transactor.TransactOptions,
) (*common.Hash, error) {
	log.Debug().
		Str("recipient", recipient.String()).
		Str("resourceID", hexutil.Encode(resourceID[:])).
		Str("amount", amount.String()).
		Uint8("destDomainID", destDomainID).
		Hex("feeData", feeData).
		Msgf("ERC20 deposit")
	var data []byte
	if opts.Priority == 0 {
		data = deposit.ConstructErc20DepositData(recipient.Bytes(), amount)
	} else {
		data = deposit.ConstructErc20DepositDataWithPriority(recipient.Bytes(), amount, opts.Priority)
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
	proposal *proposal.Proposal,
	signature []byte,
	opts transactor.TransactOptions,
) (*common.Hash, error) {
	log.Debug().
		Str("depositNonce", strconv.FormatUint(proposal.DepositNonce, 10)).
		Str("resourceID", hexutil.Encode(proposal.ResourceId[:])).
		Str("handler", proposal.HandlerAddress.String()).
		Msgf("Execute proposal")
	return c.ExecuteTransaction(
		"executeProposal",
		opts,
		proposal.Source, proposal.DepositNonce, proposal.Data, proposal.ResourceId, signature,
	)
}

func (c *BridgeContract) ExecuteProposals(
	proposals []*proposal.Proposal,
	signature []byte,
	opts transactor.TransactOptions,
) (*common.Hash, error) {
	bridgeProposals := make([]BridgeProposal, 0)
	for _, prop := range proposals {
		bridgeProposals = append(bridgeProposals, BridgeProposal{
			OriginDomainID: prop.Source,
			DepositNonce:   prop.DepositNonce,
			ResourceID:     prop.ResourceId,
			Data:           prop.Data,
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

	formattedProps := make([]interface{}, len(proposals))
	for i, prop := range proposals {
		formattedProps[i] = map[string]interface{}{
			"originDomainID": math.NewHexOrDecimal256(int64(prop.Source)),
			"depositNonce":   math.NewHexOrDecimal256(int64(prop.DepositNonce)),
			"resourceID":     hexutil.Encode(prop.ResourceId[:]),
			"data":           prop.Data,
		}
	}
	message := apitypes.TypedDataMessage{
		"proposals": formattedProps,
	}
	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"Proposal": []apitypes.Type{
				{Name: "originDomainID", Type: "uint8"},
				{Name: "depositNonce", Type: "uint64"},
				{Name: "resourceID", Type: "bytes32"},
				{Name: "data", Type: "bytes"},
			},
			"Proposals": []apitypes.Type{
				{Name: "proposals", Type: "Proposal[]"},
			},
		},
		PrimaryType: "Proposals",
		Domain: apitypes.TypedDataDomain{
			Name:              "Bridge",
			ChainId:           math.NewHexOrDecimal256(chainID.Int64()),
			Version:           "3.1.0",
			VerifyingContract: c.ContractAddress().Hex(),
		},
		Message: message,
	}

	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return []byte{}, err
	}

	typedDataHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return []byte{}, err
	}

	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(typedDataHash)))
	return crypto.Keccak256(rawData), nil
}

func (c *BridgeContract) IsProposalExecuted(p *proposal.Proposal) (bool, error) {
	log.Debug().
		Str("depositNonce", strconv.FormatUint(p.DepositNonce, 10)).
		Str("resourceID", hexutil.Encode(p.ResourceId[:])).
		Str("handler", p.HandlerAddress.String()).
		Msg("Getting is proposal executed")
	res, err := c.CallContract("isProposalExecuted", p.Source, big.NewInt(int64(p.DepositNonce)))
	if err != nil {
		return false, err
	}
	out := *abi.ConvertType(res[0], new(bool)).(*bool)
	return out, nil
}

func (c *BridgeContract) GetHandlerAddressForResourceID(
	resourceID types.ResourceID,
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
