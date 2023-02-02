// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package app

import (
	"context"
	"fmt"
	"github.com/ChainSafe/sygma-relayer/jobs"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	coreEvm "github.com/ChainSafe/chainbridge-core/chains/evm"
	coreEvents "github.com/ChainSafe/chainbridge-core/chains/evm/calls/events"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmclient"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmgaspricer"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmtransaction"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor/signAndSend"
	coreExecutor "github.com/ChainSafe/chainbridge-core/chains/evm/executor"
	coreListener "github.com/ChainSafe/chainbridge-core/chains/evm/listener"
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
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func Run() error {
	var err error

	configFlag := viper.GetString(flags.ConfigFlagName)
	configURL := viper.GetString("config-url")

	configuration := &config.Config{}
	if configURL != "" {
		configuration, err = config.GetSharedConfigFromNetwork(configURL, configuration)
		panicOnError(err)
	}

	if strings.ToLower(configFlag) == "env" {
		configuration, err = config.GetConfigFromENV(configuration)
		panicOnError(err)
	} else {
		configuration, err = config.GetConfigFromFile(configFlag, configuration)
		panicOnError(err)
	}

	logger.ConfigureLogger(configuration.RelayerConfig.LogLevel, os.Stdout)

	log.Info().Msg("Successfully loaded configuration")

	topologyProvider, err := topology.NewNetworkTopologyProvider(configuration.RelayerConfig.MpcConfig.TopologyConfiguration, http.DefaultClient)
	panicOnError(err)
	topologyStore := topology.NewTopologyStore(configuration.RelayerConfig.MpcConfig.TopologyConfiguration.Path)
	networkTopology, err := topologyStore.Topology()
	// if topology is not already in file, read from provider
	if err != nil {
		log.Debug().Msg("Reading topology from provider")
		networkTopology, err = topologyProvider.NetworkTopology()
		panicOnError(err)

		err = topologyStore.StoreTopology(networkTopology)
		panicOnError(err)
	}
	log.Info().Msg("Successfully loaded topology")

	privBytes, err := crypto.ConfigDecodeKey(configuration.RelayerConfig.MpcConfig.Key)
	panicOnError(err)

	priv, err := crypto.UnmarshalPrivateKey(privBytes)
	panicOnError(err)

	connectionGate := p2p.NewConnectionGate(networkTopology)
	host, err := p2p.NewHost(priv, networkTopology, connectionGate, configuration.RelayerConfig.MpcConfig.Port)
	panicOnError(err)
	log.Info().Str("peerID", host.ID().String()).Msg("Successfully created libp2p host")

	go health.StartHealthEndpoint(configuration.RelayerConfig.HealthPort)

	communication := p2p.NewCommunication(host, "p2p/sygma")
	electorFactory := elector.NewCoordinatorElectorFactory(host, configuration.RelayerConfig.BullyConfig)
	coordinator := tss.NewCoordinator(host, communication, electorFactory)
	keyshareStore := keyshare.NewKeyshareStore(configuration.RelayerConfig.MpcConfig.KeysharePath)

	// this is temporary solution related to specifics of aws deployment
	// effectively it waits until old instance is killed
	var db *lvldb.LVLDB
	for {
		db, err = lvldb.NewLvlDB(viper.GetString(flags.BlockstoreFlagName))
		if err != nil {
			log.Error().Err(err).Msg("Unable to connect to blockstore file, retry in 10 seconds")
			time.Sleep(10 * time.Second)
		} else {
			log.Info().Msg("Successfully connected to blockstore file")
			break
		}
	}
	blockstore := store.NewBlockStore(db)

	chains := []relayer.RelayedChain{}
	for _, chainConfig := range configuration.ChainConfigs {
		switch chainConfig["type"] {
		case "evm":
			{
				config, err := evm.NewEVMConfig(chainConfig)
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

				depositHandler := coreListener.NewETHDepositHandler(bridgeContract)
				mh := coreExecutor.NewEVMMessageHandler(bridgeContract)
				for _, handler := range config.Handlers {
					switch handler.Type {
					case "erc20":
						{
							depositHandler.RegisterDepositHandler(handler.Address, coreListener.Erc20DepositHandler)
							mh.RegisterMessageHandler(handler.Address, coreExecutor.ERC20MessageHandler)
						}
					case "permissionedGeneric":
						{
							depositHandler.RegisterDepositHandler(handler.Address, coreListener.GenericDepositHandler)
							mh.RegisterMessageHandler(handler.Address, coreExecutor.GenericMessageHandler)
						}
					case "permissionlessGeneric":
						{
							depositHandler.RegisterDepositHandler(handler.Address, listener.PermissionlessGenericDepositHandler)
							mh.RegisterMessageHandler(handler.Address, executor.PermissionlessGenericMessageHandler)
						}
					case "erc721":
						{
							depositHandler.RegisterDepositHandler(handler.Address, coreListener.Erc721DepositHandler)
							mh.RegisterMessageHandler(handler.Address, coreExecutor.ERC721MessageHandler)
						}
					}
				}
				depositListener := coreEvents.NewListener(client)
				tssListener := events.NewListener(client)
				eventHandlers := make([]coreListener.EventHandler, 0)
				l := log.With().Str("chain", fmt.Sprintf("%v", chainConfig["name"]))
				eventHandlers = append(eventHandlers, listener.NewDepositEventHandler(l, depositListener, depositHandler, bridgeAddress, *config.GeneralChainConfig.Id))
				eventHandlers = append(eventHandlers, listener.NewKeygenEventHandler(l, tssListener, coordinator, host, communication, keyshareStore, bridgeAddress, networkTopology.Threshold))
				eventHandlers = append(eventHandlers, listener.NewRefreshEventHandler(l, topologyProvider, topologyStore, tssListener, coordinator, host, communication, connectionGate, keyshareStore, bridgeAddress))
				eventHandlers = append(eventHandlers, listener.NewRetryEventHandler(l, tssListener, depositHandler, bridgeAddress, *config.GeneralChainConfig.Id, config.BlockConfirmations))
				evmListener := coreListener.NewEVMListener(client, eventHandlers, blockstore, *config.GeneralChainConfig.Id, config.BlockRetryInterval, config.BlockConfirmations, config.BlockInterval)
				executor := executor.NewExecutor(host, communication, coordinator, mh, bridgeContract, keyshareStore)

				coreEvmChain := coreEvm.NewEVMChain(evmListener, nil, blockstore, *config.GeneralChainConfig.Id, config.StartBlock, config.GeneralChainConfig.LatestBlock, config.GeneralChainConfig.FreshStart)
				chain := evm.NewEVMChain(*coreEvmChain, executor)

				chains = append(chains, chain)
			}
		default:
			panic(fmt.Errorf("type '%s' not recognized", chainConfig["type"]))
		}
	}

	go jobs.StartCommunicationHealthCheckJob(host, configuration.RelayerConfig.MpcConfig.CommHealthCheckInterval)

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
