package deploy

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/ChainSafe/sygma-core/chains/evm/calls"
	"github.com/ChainSafe/sygma-core/chains/evm/calls/contracts/erc20"
	"github.com/ChainSafe/sygma-core/chains/evm/calls/contracts/erc721"
	"github.com/ChainSafe/sygma-core/chains/evm/calls/evmclient"
	"github.com/ChainSafe/sygma-core/chains/evm/calls/evmgaspricer"
	"github.com/ChainSafe/sygma-core/chains/evm/calls/evmtransaction"
	"github.com/ChainSafe/sygma-core/chains/evm/calls/transactor/signAndSend"
	"github.com/ChainSafe/sygma-core/chains/evm/cli/flags"
	"github.com/ChainSafe/sygma-core/chains/evm/cli/logger"
	"github.com/ChainSafe/sygma-core/chains/evm/cli/utils"

	"github.com/ChainSafe/sygma/chains/evm/calls/contracts/accessControlSegregator"
	"github.com/ChainSafe/sygma/chains/evm/calls/contracts/bridge"
	"github.com/ChainSafe/sygma/chains/evm/calls/contracts/feeHandler"
	"github.com/ChainSafe/sygma/chains/evm/calls/contracts/generic"
	"github.com/ChainSafe/sygma/chains/evm/calls/util"
)

var ErrNoDeploymentFlagsProvided = errors.New("provide at least one deployment flag. For help use --help")

var DeployEVM = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy smart contracts",
	Long:  "This command can be used to deploy all or some of the contracts required for bridging. Selection of contracts can be made by either specifying --all or a subset of flags",
	PreRun: func(cmd *cobra.Command, args []string) {
		logger.LoggerMetadata(cmd.Name(), cmd.Flags())
	},
	RunE: CallDeployCLI,
	Args: func(cmd *cobra.Command, args []string) error {
		err := ValidateDeployFlags(cmd, args)
		if err != nil {
			return err
		}
		err = ProcessDeployFlags(cmd, args)
		return err
	},
}

const (
	TwoDaysTermInBlocks int64 = 6200
)

func BindDeployEVMFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&Bridge, "bridge", false, "Deploy bridge")
	cmd.Flags().StringVar(&BridgeAddress, "bridge-address", "", "Bridge contract address. Should be provided if handlers are deployed separately")
	cmd.Flags().BoolVar(&DeployAll, "all", false, "Deploy all")
	cmd.Flags().Uint8Var(&DomainId, "domain", 1, "Domain ID for the instance")
	cmd.Flags().BoolVar(&Erc20, "erc20", false, "Deploy ERC20")
	cmd.Flags().BoolVar(&Erc20Handler, "erc20-handler", false, "Deploy ERC20 handler")
	cmd.Flags().StringVar(&Erc20Name, "erc20-name", "", "ERC20 contract name")
	cmd.Flags().StringVar(&Erc20Symbol, "erc20-symbol", "", "ERC20 contract symbol")
	cmd.Flags().BoolVar(&Erc721, "erc721", false, "Deploy ERC721")
	cmd.Flags().BoolVar(&Erc721Handler, "erc721-handler", false, "Deploy ERC721 handler")
	cmd.Flags().StringVar(&Erc721Name, "erc721-name", "", "ERC721 contract name")
	cmd.Flags().StringVar(&Erc721Symbol, "erc721-symbol", "", "ERC721 contract symbol")
	cmd.Flags().StringVar(&Erc721BaseURI, "erc721-base-uri", "", "ERC721 base URI")
	cmd.Flags().BoolVar(&GenericHandler, "generic-handler", false, "Deploy generic handler")
	cmd.Flags().BoolVar(&FeeHandlerWithOracle, "fee-handler-with-oracle", false, "Deploy fee handler with fee oracle. The basic fee handler will be deployed by default")
	cmd.Flags().StringSliceVar(&Relayers, "relayers", []string{}, "List of initial relayers")
	cmd.Flags().StringSliceVar(
		&Admins,
		"admins",
		[]string{"0xBDC18Db00cEd96B2E7356613383eFE02a7183AB4", "0xBDC18Db00cEd96B2E7356613383eFE02a7183AB4", "0xBDC18Db00cEd96B2E7356613383eFE02a7183AB4", "0xBDC18Db00cEd96B2E7356613383eFE02a7183AB4", "0xBDC18Db00cEd96B2E7356613383eFE02a7183AB4", "0xBDC18Db00cEd96B2E7356613383eFE02a7183AB4", "0xBDC18Db00cEd96B2E7356613383eFE02a7183AB4", "0xBDC18Db00cEd96B2E7356613383eFE02a7183AB4", "0xBDC18Db00cEd96B2E7356613383eFE02a7183AB4", "0xBDC18Db00cEd96B2E7356613383eFE02a7183AB4", "0xBDC18Db00cEd96B2E7356613383eFE02a7183AB4", "0xBDC18Db00cEd96B2E7356613383eFE02a7183AB4", "0xBDC18Db00cEd96B2E7356613383eFE02a7183AB4", "0xBDC18Db00cEd96B2E7356613383eFE02a7183AB4"},
		"List of initial admin addresses per admin function",
	)
	cmd.Flags().StringSliceVar(
		&AdminFunctions,
		"admin-functions",
		[]string{"80ae1c28", "ffaac0eb", "cb10f215", "5a1ad87c", "8c0c2631", "edc20c3c", "d15ef64e", "9d33b6d4", "8b63aebf", "bd2a1820", "6ba6db6b", "d2e5fae9", "f5f63b39", "a973ec93"},
		"List of initial admin functions in non-prefixed hex format. By default al functions are included",
	)
	cmd.Flags().Uint64Var(&RelayerThreshold, "relayer-threshold", 1, "Number of votes required for a proposal to pass")
}

func init() {
	BindDeployEVMFlags(DeployEVM)
}

func ValidateDeployFlags(cmd *cobra.Command, args []string) error {
	if len(Admins) != len(AdminFunctions) {
		return fmt.Errorf("admins and admin functions length should be equal")
	}

	Deployments = make([]string, 0)
	if DeployAll {
		flags.MarkFlagsAsRequired(cmd, "relayer-threshold", "domain", "erc20-symbol", "erc20-name")
		Deployments = append(Deployments, []string{"bridge", "erc20-handler", "erc721-handler", "generic-handler", "erc20", "erc721"}...)
	} else {
		if Bridge {
			flags.MarkFlagsAsRequired(cmd, "relayer-threshold", "domain")
			Deployments = append(Deployments, "bridge")
		}
		if Erc20Handler {
			if !Bridge {
				flags.MarkFlagsAsRequired(cmd, "bridge-address")
			}
			Deployments = append(Deployments, "erc20-handler")
		}
		if Erc721Handler {
			if !Bridge {
				flags.MarkFlagsAsRequired(cmd, "bridge-address")
			}
			Deployments = append(Deployments, "erc721-handler")
		}
		if GenericHandler {
			if !Bridge {
				flags.MarkFlagsAsRequired(cmd, "bridge-address")
			}
			Deployments = append(Deployments, "generic-handler")
		}
		if Erc20 {
			flags.MarkFlagsAsRequired(cmd, "erc20-symbol", "erc20-name")
			Deployments = append(Deployments, "erc20")
		}
		if Erc721 {
			flags.MarkFlagsAsRequired(cmd, "erc721-name", "erc721-symbol", "erc721-base-uri")
			Deployments = append(Deployments, "erc721")
		}
	}

	if len(Deployments) == 0 {
		log.Error().Err(ErrNoDeploymentFlagsProvided)
		return ErrNoDeploymentFlagsProvided
	}

	return nil
}

var Deployments []string
var BridgeAddr common.Address
var RelayerAddresses []common.Address

func ProcessDeployFlags(cmd *cobra.Command, args []string) error {
	if common.IsHexAddress(BridgeAddress) {
		BridgeAddr = common.HexToAddress(BridgeAddress)
	}
	for _, addr := range Relayers {
		if !common.IsHexAddress(addr) {
			return fmt.Errorf("invalid relayer address %s", addr)
		}
		RelayerAddresses = append(RelayerAddresses, common.HexToAddress(addr))
	}
	return nil
}

func CallDeployCLI(cmd *cobra.Command, args []string) error {
	txFabric := evmtransaction.NewTransaction
	return DeployCLI(cmd, args, txFabric, &evmgaspricer.LondonGasPriceDeterminant{})
}

func DeployCLI(cmd *cobra.Command, args []string, txFabric calls.TxFabric, gasPricer utils.GasPricerWithPostConfig) error {
	// fetch global flag values
	url, gasLimit, gasPrice, senderKeyPair, _, err := flags.GlobalFlagValues(cmd)
	if err != nil {
		return err
	}

	log.Debug().Msgf("url: %s gas limit: %v gas price: %v", url, gasLimit, gasPrice)
	log.Debug().Msgf("SENDER Private key 0x%s", hex.EncodeToString(crypto.FromECDSA(senderKeyPair.PrivateKey())))

	ethClient, err := evmclient.NewEVMClient(url, senderKeyPair.PrivateKey())
	if err != nil {
		log.Error().Err(fmt.Errorf("ethereum client error: %v", err)).Msg("error initializing new EVM client")
		return err
	}
	gasPricer.SetClient(ethClient)
	gasPricer.SetOpts(&evmgaspricer.GasPricerOpts{UpperLimitFeePerGas: gasPrice})
	log.Debug().Msgf("Relayers for deploy %+v", Relayers)
	log.Debug().Msgf("all bool: %v", DeployAll)

	t := signAndSend.NewSignAndSendTransactor(txFabric, gasPricer, ethClient)

	deployedContracts := make(map[string]string)
	for _, v := range Deployments {
		switch v {
		case "bridge":
			admins := make([]common.Address, len(Admins))
			adminFunctions := make([][4]byte, len(AdminFunctions))
			for i, functionHex := range AdminFunctions {
				admins[i] = ethClient.From()
				hexBytes, _ := hex.DecodeString(string(functionHex))
				adminFunctions[i] = util.SliceTo4Bytes(hexBytes)
			}
			accessControlSegregatorContract := accessControlSegregator.NewAccessControlSegregatorContract(ethClient, common.Address{}, t)
			_, err := accessControlSegregatorContract.DeployContract(
				AdminFunctions,
				Admins,
			)
			if err != nil {
				log.Error().Err(fmt.Errorf("access control segregator deploy failed: %w", err))
				return err
			}

			log.Debug().Msgf("deploying bridge..")
			bc := bridge.NewBridgeContract(ethClient, common.Address{}, t)
			BridgeAddr, err = bc.DeployContract(DomainId, accessControlSegregatorContract.ContractAddress())
			if err != nil {
				log.Error().Err(fmt.Errorf("bridge deploy failed: %w", err))
				return err
			}
			deployedContracts["bridge"] = BridgeAddr.String()
			log.Debug().Msgf("bridge address; %v", BridgeAddr.String())

			// fee handler deployment
			feeHandlerAddress := common.Address{}
			if FeeHandlerWithOracle {
				// deploy fee handler with oracle
				fhwo := feeHandler.NewFeeHandlerWithOracleContract(ethClient, common.Address{}, t)
				feeHandlerWithOracleAddress, err := fhwo.DeployContract(BridgeAddr)
				if err != nil {
					log.Error().Err(fmt.Errorf("fee handler(with oracle) deploy failed: %w", err))
					return err
				}
				feeHandlerAddress = feeHandlerWithOracleAddress
			} else {
				// deploy basic fee handler, init fee will be 0
				bfh := feeHandler.NewBasicFeeHandlerContract(ethClient, common.Address{}, t)
				basicFeeHandlerAddress, err := bfh.DeployContract(BridgeAddr)
				if err != nil {
					log.Error().Err(fmt.Errorf("basic fee handler deploy failed: %w", err))
					return err
				}
				feeHandlerAddress = basicFeeHandlerAddress
			}

			deployedContracts["feeHandler"] = feeHandlerAddress.String()
			log.Debug().Msgf("fee handler address; %v", feeHandlerAddress.String())
		case "erc20":
			log.Debug().Msgf("deploying ERC20..")
			erc20Contract := erc20.NewERC20Contract(ethClient, common.Address{}, t)
			erc20Addr, err := erc20Contract.DeployContract(Erc20Name, Erc20Symbol)
			if err != nil {
				log.Error().Err(fmt.Errorf("erc 20 deploy failed: %w", err))
				return err
			}
			deployedContracts["erc20Token"] = erc20Addr.String()
		case "erc20-handler":
			log.Debug().Msgf("deploying ERC20 handler..")
			erc20HandlerContract := erc20.NewERC20HandlerContract(ethClient, common.Address{}, t)
			erc20HandlerAddr, err := erc20HandlerContract.DeployContract(BridgeAddr)
			if err != nil {
				log.Error().Err(fmt.Errorf("ERC20 handler deploy failed: %w", err))
				return err
			}
			deployedContracts["erc20Handler"] = erc20HandlerAddr.String()
		case "erc721":
			log.Debug().Msgf("deploying ERC721..")
			erc721Contract := erc721.NewErc721Contract(ethClient, common.Address{}, t)
			erc721Addr, err := erc721Contract.DeployContract(Erc721Name, Erc721Symbol, Erc721BaseURI)
			if err != nil {
				log.Error().Err(fmt.Errorf("ERC721 deploy failed: %w", err))
				return err
			}
			deployedContracts["erc721Token"] = erc721Addr.String()
		case "erc721-handler":
			log.Debug().Msgf("deploying ERC721 handler..")
			erc721HandlerContract := erc721.NewERC721HandlerContract(ethClient, common.Address{}, t)
			erc721HandlerAddr, err := erc721HandlerContract.DeployContract(BridgeAddr)
			if err != nil {
				log.Error().Err(fmt.Errorf("ERC721 handler deploy failed: %w", err))
				return err
			}
			deployedContracts["erc721Handler"] = erc721HandlerAddr.String()
		case "generic-handler":
			log.Debug().Msgf("deploying generic handler..")
			emptyAddr := common.Address{}
			if BridgeAddr == emptyAddr {
				log.Error().Err(errors.New("bridge flag or bridge-address param should be set for contracts Deployments"))
				return err
			}
			genericHandlerContract := generic.NewGenericHandlerContract(ethClient, common.Address{}, t)
			genericHandlerAddr, err := genericHandlerContract.DeployContract(BridgeAddr)
			if err != nil {
				log.Error().Err(fmt.Errorf("generic handler deploy failed: %w", err))
				return err
			}
			deployedContracts["genericHandler"] = genericHandlerAddr.String()
		}
	}
	fmt.Printf(`
	Deployed contracts
=========================================================
Bridge: %s
---------------------------------------------------------
ERC20 Token: %s
---------------------------------------------------------
ERC20 Handler: %s
---------------------------------------------------------
ERC721 Token: %s
---------------------------------------------------------
ERC721 Handler: %s
---------------------------------------------------------
Generic Handler: %s
---------------------------------------------------------
Fee Handler: %s
=========================================================
	`,
		deployedContracts["bridge"],
		deployedContracts["erc20Token"],
		deployedContracts["erc20Handler"],
		deployedContracts["erc721Token"],
		deployedContracts["erc721Handler"],
		deployedContracts["genericHandler"],
		deployedContracts["feeHandler"],
	)
	return nil
}
