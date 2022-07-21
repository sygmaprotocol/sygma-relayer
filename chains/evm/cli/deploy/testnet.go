package deploy

import (
	"errors"

	"github.com/ChainSafe/sygma-core/chains/evm/cli/flags"
	"github.com/ChainSafe/sygma-core/chains/evm/cli/logger"
	"github.com/ChainSafe/sygma-core/util"
	"github.com/ChainSafe/sygma/chains/evm/calls/deployutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
)

var deplotTestnet = &cobra.Command{
	Use:   "testnet",
	Short: "deploy testnet Sygma copy",
	Long:  "Lighter version of deploy CLI that deploys exact version of Sygma bridge that has been deployed on Testnet",
	PreRun: func(cmd *cobra.Command, args []string) {
		logger.LoggerMetadata(cmd.Name(), cmd.Flags())
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return util.CallPersistentPreRun(cmd, args)
	},
	Args: func(cmd *cobra.Command, args []string) error {
		err := ValidateTestNetDeployFlags(cmd, args)
		if err != nil {
			return err
		}

		err = ProcessTestNetDeployFlags(cmd, args)
		return err
	},
	RunE: deployTestnet,
}

func ValidateTestNetDeployFlags(cmd *cobra.Command, args []string) error {
	if !common.IsHexAddress(FeeOracleAddressStr) {
		return errors.New("invalid feeoracle address")
	}
	return nil
}

func ProcessTestNetDeployFlags(cmd *cobra.Command, args []string) error {
	FeeOracleAddress = common.HexToAddress(FeeOracleAddressStr)
	return nil
}

func BindSDeployTestNetFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&FeeOracleAddressStr, "feeOracle", "", "Fee router")
	cmd.Flags().Uint8Var(&DomainId, "domainID", 0, "FeeHandler will be used for this destDomainID")
	cmd.Flags().StringVar(&ResourceID, "resourceID", "", "FeeHander will be used for this resourceID")
	cmd.Flags().StringVar(&Erc20Name, "erc20Symbol", "", "erc20Symbol")
	cmd.Flags().StringVar(&Erc20Symbol, "erc20Name", "", "erc20 name")
	cmd.Flags().Uint16Var(&FeePercent, "feePercent", 0, "set this fee percent on FeeHandler 100=1%")
	cmd.Flags().Uint32Var(&FeeGasUsed, "feeGas", 100000, "Gas used on the destination network for current resourceID")

	flags.MarkFlagsAsRequired(cmd, "feeOracle", "domainID", "resourceID")
}

func init() {
	BindSDeployTestNetFlags(deplotTestnet)
}

func deployTestnet(cmd *cobra.Command, args []string) error {

	dr, err := deployutils.SetupSygma(&deployutils.DeployConfig{
		DeployKey:        senderKeyPair.PrivateKey(),
		NodeURL:          url,
		DomainID:         DomainId,
		ResourceID:       ResourceID,
		FeeOracleAddress: FeeOracleAddress,
		Erc20Symbol:      Erc20Symbol,
		Erc20Name:        Erc20Name,
		FeePercent:       FeePercent,
		FeeGas:           FeeGasUsed,
	})
	if err != nil {
		return err
	}
	dr.PrettyPrintDeployResutls()
	return nil
}
