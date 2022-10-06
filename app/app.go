// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package app

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ChainSafe/sygma-relayer/comm"

	coreEvm "github.com/ChainSafe/chainbridge-core/chains/evm"
	coreEvents "github.com/ChainSafe/chainbridge-core/chains/evm/calls/events"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmclient"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmgaspricer"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmtransaction"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor/signAndSend"
	coreExecutor "github.com/ChainSafe/chainbridge-core/chains/evm/executor"
	coreListener "github.com/ChainSafe/chainbridge-core/chains/evm/listener"
	"github.com/ChainSafe/chainbridge-core/config/chain"
	"github.com/ChainSafe/chainbridge-core/flags"
	"github.com/ChainSafe/chainbridge-core/logger"
	"github.com/ChainSafe/chainbridge-core/lvldb"
	"github.com/ChainSafe/chainbridge-core/opentelemetry"
	"github.com/ChainSafe/chainbridge-core/relayer"
	"github.com/ChainSafe/chainbridge-core/store"
	"github.com/ChainSafe/sygma-relayer/chains/evm"
	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/contracts/bridge"
	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/events"
	"github.com/ChainSafe/sygma-relayer/chains/evm/executor"
	"github.com/ChainSafe/sygma-relayer/chains/evm/listener"
	"github.com/ChainSafe/sygma-relayer/comm/elector"
	"github.com/ChainSafe/sygma-relayer/comm/p2p"
	"github.com/ChainSafe/sygma-relayer/config"
	"github.com/ChainSafe/sygma-relayer/health"
	"github.com/ChainSafe/sygma-relayer/keyshare"
	"github.com/ChainSafe/sygma-relayer/topology"
	"github.com/ChainSafe/sygma-relayer/tss"
	"github.com/ethereum/go-ethereum/common"
	secp256k1 "github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func Run() error {
	var err error
	var configuration config.Config

	configFlag := viper.GetString(flags.ConfigFlagName)
	if strings.ToLower(configFlag) == "env" {
		configuration, err = config.GetConfigFromENV()
		panicOnError(err)
	} else {
		configuration, err = config.GetConfigFromFile(configFlag)
		panicOnError(err)
	}

	logger.ConfigureLogger(configuration.RelayerConfig.LogLevel, os.Stdout)

	go health.StartHealthEndpoint(configuration.RelayerConfig.HealthPort)

	topologyProvider, err := topology.NewNetworkTopologyProvider(configuration.RelayerConfig.MpcConfig.TopologyConfiguration)
	panicOnError(err)
	topologyStore := topology.NewTopologyStore(configuration.RelayerConfig.MpcConfig.TopologyConfiguration.Path)
	networkTopology, err := topologyStore.Topology()
	// if topology is not already in file, read from provider
	if err != nil {
		networkTopology, err = topologyProvider.NetworkTopology()
		panicOnError(err)

		err = topologyStore.StoreTopology(networkTopology)
		panicOnError(err)
	}

	// this is temporary solution related to specifics of aws deployment
	// effectively it waits until old instance is killed
	var db *lvldb.LVLDB
	for {
		db, err = lvldb.NewLvlDB(viper.GetString(flags.BlockstoreFlagName))
		if err != nil {
			time.Sleep(5 * time.Second)
		} else {
			log.Info().Msg("Successfully connected to blockstore file")
			break
		}
	}

	blockstore := store.NewBlockStore(db)

	privBytes, err := crypto.ConfigDecodeKey(configuration.RelayerConfig.MpcConfig.Key)
	panicOnError(err)

	priv, err := crypto.UnmarshalPrivateKey(privBytes)
	panicOnError(err)

	connectionGate := p2p.NewConnectionGate(networkTopology)
	host, err := p2p.NewHost(priv, networkTopology, connectionGate, configuration.RelayerConfig.MpcConfig.Port)
	panicOnError(err)

	healthComm := p2p.NewCommunication(host, "p2p/health")
	go comm.ExecuteCommHealthCheck(healthComm, host.Peerstore().Peers())

	communication := p2p.NewCommunication(host, "p2p/sygma")
	electorFactory := elector.NewCoordinatorElectorFactory(host, configuration.RelayerConfig.BullyConfig)
	coordinator := tss.NewCoordinator(host, communication, electorFactory)
	keyshareStore := keyshare.NewKeyshareStore(configuration.RelayerConfig.MpcConfig.KeysharePath)

	chains := []relayer.RelayedChain{}
	for _, chainConfig := range configuration.ChainConfigs {
		switch chainConfig["type"] {
		case "evm":
			{
				config, err := chain.NewEVMConfig(chainConfig)
				panicOnError(err)

				privateKey, err := secp256k1.HexToECDSA(config.GeneralChainConfig.Key)
				panicOnError(err)

				client, err := evmclient.NewEVMClient(config.GeneralChainConfig.Endpoint, privateKey)
				panicOnError(err)

				mod := big.NewInt(0).Mod(config.StartBlock, config.BlockConfirmations)
				// startBlock % blockConfirmations == 0
				if mod.Cmp(big.NewInt(0)) != 0 {
					config.StartBlock.Sub(config.StartBlock, mod)
				}

				bridgeAddress := common.HexToAddress(config.Bridge)
				gasPricer := evmgaspricer.NewLondonGasPriceClient(client, &evmgaspricer.GasPricerOpts{
					UpperLimitFeePerGas: config.MaxGasPrice,
					GasPriceFactor:      config.GasMultiplier,
				})
				t := signAndSend.NewSignAndSendTransactor(evmtransaction.NewTransaction, gasPricer, client)
				bridgeContract := bridge.NewBridgeContract(client, bridgeAddress, t)

				pGenericHandler := chainConfig["permissionlessGenericHandler"].(string)
				depositHandler := coreListener.NewETHDepositHandler(bridgeContract)
				depositHandler.RegisterDepositHandler(config.Erc20Handler, coreListener.Erc20DepositHandler)
				depositHandler.RegisterDepositHandler(config.Erc721Handler, coreListener.Erc721DepositHandler)
				depositHandler.RegisterDepositHandler(config.GenericHandler, coreListener.GenericDepositHandler)
				depositHandler.RegisterDepositHandler(pGenericHandler, listener.PermissionlessGenericDepositHandler)
				depositListener := coreEvents.NewListener(client)
				tssListener := events.NewListener(client)
				eventHandlers := make([]coreListener.EventHandler, 0)
				eventHandlers = append(eventHandlers, coreListener.NewDepositEventHandler(depositListener, depositHandler, bridgeAddress, *config.GeneralChainConfig.Id))
				eventHandlers = append(eventHandlers, listener.NewKeygenEventHandler(tssListener, coordinator, host, communication, keyshareStore, bridgeAddress, networkTopology.Threshold))
				eventHandlers = append(eventHandlers, listener.NewRefreshEventHandler(topologyProvider, topologyStore, tssListener, coordinator, host, communication, connectionGate, keyshareStore, bridgeAddress))
				eventHandlers = append(eventHandlers, listener.NewRetryEventHandler(tssListener, depositHandler, bridgeAddress, *config.GeneralChainConfig.Id, config.BlockConfirmations))
				evmListener := coreListener.NewEVMListener(client, eventHandlers, blockstore, config)

				mh := coreExecutor.NewEVMMessageHandler(bridgeContract)
				mh.RegisterMessageHandler(config.Erc20Handler, coreExecutor.ERC20MessageHandler)
				mh.RegisterMessageHandler(config.Erc721Handler, coreExecutor.ERC721MessageHandler)
				mh.RegisterMessageHandler(config.GenericHandler, coreExecutor.GenericMessageHandler)
				mh.RegisterMessageHandler(pGenericHandler, executor.PermissionlessGenericMessageHandler)
				executor := executor.NewExecutor(host, communication, coordinator, mh, bridgeContract, keyshareStore)

				coreEvmChain := coreEvm.NewEVMChain(evmListener, nil, blockstore, config)
				chain := evm.NewEVMChain(*coreEvmChain, executor)

				chains = append(chains, chain)
			}
		default:
			panic(fmt.Errorf("type '%s' not recognized", chainConfig["type"]))
		}
	}

	r := relayer.NewRelayer(
		chains,
		&opentelemetry.ConsoleTelemetry{},
	)

	errChn := make(chan error)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Start(ctx, errChn)

	sysErr := make(chan os.Signal, 1)
	signal.Notify(sysErr,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGHUP,
		syscall.SIGQUIT)

	relayerName := viper.GetString("name")
	log.Info().Msgf("Started relayer: %s with PID: %s", relayerName, host.ID().Pretty())

	select {
	case err := <-errChn:
		log.Error().Err(err).Msg("failed to listen and serve")
		return err
	case sig := <-sysErr:
		log.Info().Msgf("terminating got ` [%v] signal", sig)
		return nil
	}
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
