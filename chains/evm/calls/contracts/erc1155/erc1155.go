package erc1155

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/contracts"
	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/consts"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

type ERC1155Contract struct {
	contracts.Contract
}

func NewErc1155Contract(
	client calls.ContractCallerDispatcher,
	erc1155ContractAddress common.Address,
	t transactor.Transactor,
) *ERC1155Contract {
	a, _ := abi.JSON(strings.NewReader(consts.ERC1155PresetMinterPauserABI))
	b := common.FromHex(consts.ERC1155PresetMinterPauserABI)
	return &ERC1155Contract{contracts.NewContract(erc1155ContractAddress, a, b, client, t)}
}

func (c *ERC1155Contract) Approve(
	tokenId *big.Int, recipient common.Address, opts transactor.TransactOptions,
) (*common.Hash, error) {
	log.Debug().Msgf("Approving %s token for %s", tokenId.String(), recipient.String())
	return c.ExecuteTransaction("setApprovalForAll", opts, recipient, true)
}

func (c *ERC1155Contract) Mint(
	tokenId *big.Int, amount *big.Int, metadata []byte, destination common.Address, opts transactor.TransactOptions,
) (*common.Hash, error) {
	log.Debug().Msgf("Minting tokens %s to %s", tokenId.String(), destination.String())
	return c.ExecuteTransaction("mint", opts, destination, tokenId, amount, metadata)
}

func (c *ERC1155Contract) BalanceOf(account common.Address, id *big.Int) (*big.Int, error) {
	res, err := c.CallContract("balanceOf", account, id)
	if err != nil {
		return nil, err
	}

	amount := abi.ConvertType(res[0], new(big.Int)).(*big.Int)
	fmt.Println(amount)
	return amount, nil
}
