package local

import (
	"fmt"

	"github.com/ChainSafe/sygma-core/chains/evm/calls/evmclient"
	"github.com/ChainSafe/sygma-core/chains/evm/calls/evmtransaction"
	"github.com/ChainSafe/sygma-core/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
)

var LocalSetupCmd = &cobra.Command{
	Use:   "local-setup",
	Short: "Deploy and prefund a local bridge for testing",
	Long:  "The local-setup command deploys a bridge, ERC20, ERC721 and generic handler contracts with preconfigured accounts and appropriate handlers",
	RunE:  localSetup,
}

// configuration
var (
	ethEndpoint1 = "ws://localhost:8546"
	ethEndpoint2 = "ws://localhost:8548"
	fabric1      = evmtransaction.NewTransaction
	fabric2      = evmtransaction.NewTransaction
)

func BindLocalSetupFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&ethEndpoint1, "endpoint1", "", "RPC endpoint of the first network")
	cmd.Flags().StringVar(&ethEndpoint2, "endpoint2", "", "RPC endpoint of the second network")
}

func init() {
	BindLocalSetupFlags(LocalSetupCmd)
}

func localSetup(cmd *cobra.Command, args []string) error {
	// init client1
	ethClient, err := evmclient.NewEVMClient(ethEndpoint1, CharlieKp.PrivateKey())
	if err != nil {
		return err
	}

	// init client2
	ethClient2, err := evmclient.NewEVMClient(ethEndpoint2, CharlieKp.PrivateKey())
	if err != nil {
		return err
	}

	// chain 1
	// domainsId: 0
	config, err := SetupEVMBridge(ethClient, fabric1, 1, CharlieKp.CommonAddress())
	if err != nil {
		return err
	}

	// chain 2
	// domainId: 1
	config2, err := SetupEVMBridge(ethClient2, fabric2, 2, CharlieKp.CommonAddress())
	if err != nil {
		return err
	}

	prettyPrint(config, config2)

	return nil
}

func prettyPrint(config, config2 BridgeConfig) {
	fmt.Printf(`
===============================================
🎉🎉🎉 ChainBridge Successfully Deployed 🎉🎉🎉

- Chain 1 -
%s

- Chain 2 -
%s

===============================================
`,
		prettyFormatChainInfo(config),
		prettyFormatChainInfo(config2),
	)
}

func prettyFormatChainInfo(cfg BridgeConfig) string {
	return fmt.Sprintf(`
Bridge: %s
Fee Handler: %s (is basic fee handler: %t, fee amount: %v wei)
ERC20: %s
ERC20LockRelease: %s
ERC20 Handler: %s
ERC721: %s
ERC721 Handler: %s
Generic Handler: %s
Asset Store: %s
ERC20 resourceId: %s
ERC20LockRelease resourceId: %s
ERC721 resourceId: %s
Generic resourceId: %s
`,
		cfg.BridgeAddr,
		cfg.FeeHandlerAddr,
		cfg.IsBasicFeeHandler,
		cfg.Fee,
		cfg.Erc20Addr,
		cfg.Erc20LockReleaseAddr,
		cfg.Erc20HandlerAddr,
		cfg.Erc721Addr,
		cfg.Erc721HandlerAddr,
		cfg.GenericHandlerAddr,
		cfg.AssetStoreAddr,
		rIDtoString(cfg.Erc20ResourceID),
		rIDtoString(cfg.Erc20LockReleaseResourceID),
		rIDtoString(cfg.Erc721ResourceID),
		rIDtoString(cfg.GenericResourceID),
	)
}

func rIDtoString(rid types.ResourceID) string {
	return common.Bytes2Hex(rid[:])
}
