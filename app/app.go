// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	coreEvents "github.com/ChainSafe/chainbridge-core/chains/evm/calls/events"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmclient"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmgaspricer"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmtransaction"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor/monitored"
	coreExecutor "github.com/ChainSafe/chainbridge-core/chains/evm/executor"
	coreListener "github.com/ChainSafe/chainbridge-core/chains/evm/listener"
	"github.com/ChainSafe/chainbridge-core/crypto/secp256k1"
	"github.com/ChainSafe/chainbridge-core/flags"
	"github.com/ChainSafe/chainbridge-core/logger"
	"github.com/ChainSafe/chainbridge-core/lvldb"
	"github.com/ChainSafe/chainbridge-core/opentelemetry"
	"github.com/ChainSafe/chainbridge-core/relayer"
	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/ChainSafe/chainbridge-core/store"
	"github.com/ChainSafe/sygma-relayer/chains/evm"
	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/contracts/bridge"
	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/events"
	"github.com/ChainSafe/sygma-relayer/chains/evm/executor"
	"github.com/ChainSafe/sygma-relayer/chains/evm/listener"
	"github.com/ChainSafe/sygma-relayer/chains/substrate"
	"github.com/ChainSafe/sygma-relayer/chains/substrate/client"
	"github.com/ChainSafe/sygma-relayer/chains/substrate/connection"
	substrateExecutor "github.com/ChainSafe/sygma-relayer/chains/substrate/executor"
	substrate_listener "github.com/ChainSafe/sygma-relayer/chains/substrate/listener"
	substrate_pallet "github.com/ChainSafe/sygma-relayer/chains/substrate/pallet"
	"github.com/ChainSafe/sygma-relayer/metrics"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"

	"github.com/ChainSafe/sygma-relayer/comm/elector"
	"github.com/ChainSafe/sygma-relayer/comm/p2p"
	"github.com/ChainSafe/sygma-relayer/config"
	"github.com/ChainSafe/sygma-relayer/health"
	"github.com/ChainSafe/sygma-relayer/jobs"
	"github.com/ChainSafe/sygma-relayer/keyshare"
	"github.com/ChainSafe/sygma-relayer/topology"
	"github.com/ChainSafe/sygma-relayer/tss"
	"github.com/ethereum/go-ethereum/common"
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
		networkTopology, err = topologyProvider.NetworkTopology("")
		panicOnError(err)

		err = topologyStore.StoreTopology(networkTopology)
		panicOnError(err)
	}
	log.Info().Msgf("Successfully loaded topology")

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

	// wait until executions are done and then stop further executions before exiting
	exitLock := &sync.RWMutex{}
	defer exitLock.Lock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	chains := []relayer.RelayedChain{}
	for _, chainConfig := range configuration.ChainConfigs {
		switch chainConfig["type"] {
		case "evm":
			{
				config, err := evm.NewEVMConfig(chainConfig)
				panicOnError(err)

				kp, err := secp256k1.NewKeypairFromString(config.GeneralChainConfig.Key)
				panicOnError(err)

				client, err := evmclient.NewEVMClient(config.GeneralChainConfig.Endpoint, kp)
				panicOnError(err)

				log.Info().Str("domain", config.String()).Msgf("Registering EVM domain")

				bridgeAddress := common.HexToAddress(config.Bridge)
				gasPricer := evmgaspricer.NewLondonGasPriceClient(client, &evmgaspricer.GasPricerOpts{
					UpperLimitFeePerGas: config.MaxGasPrice,
					GasPriceFactor:      config.GasMultiplier,
				})
				t := monitored.NewMonitoredTransactor(evmtransaction.NewTransaction, gasPricer, client, config.MaxGasPrice, config.GasIncreasePercentage)
				go t.Monitor(ctx, time.Minute*3, time.Minute*10, time.Minute)
				bridgeContract := bridge.NewBridgeContract(client, bridgeAddress, t)

				depositHandler := coreListener.NewETHDepositHandler(bridgeContract)
				mh := coreExecutor.NewEVMMessageHandler(bridgeContract)
				for _, handler := range config.Handlers {
					switch handler.Type {
					case "erc20":
						{
							depositHandler.RegisterDepositHandler(handler.Address, listener.Erc20DepositHandler)
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
				l := log.With().Str("chain", fmt.Sprintf("%v", config.GeneralChainConfig.Name)).Uint8("domainID", *config.GeneralChainConfig.Id)
				eventHandlers = append(eventHandlers, listener.NewDepositEventHandler(l, depositListener, depositHandler, bridgeAddress, *config.GeneralChainConfig.Id))
				eventHandlers = append(eventHandlers, listener.NewKeygenEventHandler(l, tssListener, coordinator, host, communication, keyshareStore, bridgeAddress, networkTopology.Threshold))
				eventHandlers = append(eventHandlers, listener.NewRefreshEventHandler(l, topologyProvider, topologyStore, tssListener, coordinator, host, communication, connectionGate, keyshareStore, bridgeAddress))
				eventHandlers = append(eventHandlers, listener.NewRetryEventHandler(l, tssListener, depositHandler, bridgeAddress, *config.GeneralChainConfig.Id, config.BlockConfirmations))
				evmListener := coreListener.NewEVMListener(client, eventHandlers, blockstore, *config.GeneralChainConfig.Id, config.BlockRetryInterval, config.BlockConfirmations, config.BlockInterval)
				executor := executor.NewExecutor(host, communication, coordinator, mh, bridgeContract, keyshareStore, exitLock)

				chain := evm.NewEVMChain(
					client, evmListener, executor, blockstore, *config.GeneralChainConfig.Id, config.StartBlock,
					config.BlockInterval, config.GeneralChainConfig.FreshStart, config.GeneralChainConfig.LatestBlock,
				)

				chains = append(chains, chain)
			}
		case "substrate":
			{
				config, err := substrate.NewSubstrateConfig(chainConfig)
				if err != nil {
					panic(err)
				}

				conn, err := connection.NewSubstrateConnection(config.GeneralChainConfig.Endpoint)
				if err != nil {
					panic(err)
				}
				keyPair, err := signature.KeyringPairFromSecret(config.GeneralChainConfig.Key, config.SubstrateNetwork)
				if err != nil {
					panic(err)
				}
				substrateClient := client.NewSubstrateClient(conn, &keyPair, config.ChainID, config.Tip)
				bridgePallet := substrate_pallet.NewPallet(substrateClient)

				log.Info().Str("domain", config.String()).Msgf("Registering substrate domain")

				l := log.With().Str("chain", fmt.Sprintf("%v", config.GeneralChainConfig.Name)).Uint8("domainID", *config.GeneralChainConfig.Id)
				depositHandler := substrate_listener.NewSubstrateDepositHandler()
				depositHandler.RegisterDepositHandler(message.FungibleTransfer, substrate_listener.FungibleTransferHandler)
				eventHandlers := make([]substrate_listener.EventHandler, 0)
				eventHandlers = append(eventHandlers, substrate_listener.NewFungibleTransferEventHandler(l, *config.GeneralChainConfig.Id, depositHandler))
				eventHandlers = append(eventHandlers, substrate_listener.NewRetryEventHandler(l, conn, depositHandler, *config.GeneralChainConfig.Id))
				substrateListener := substrate_listener.NewSubstrateListener(conn, eventHandlers, config)

				mh := substrateExecutor.NewSubstrateMessageHandler()
				mh.RegisterMessageHandler(message.FungibleTransfer, substrateExecutor.FungibleTransferMessageHandler)

				sExecutor := substrateExecutor.NewExecutor(host, communication, coordinator, mh, bridgePallet, keyshareStore, conn, exitLock)
				substrateChain := substrate.NewSubstrateChain(substrateClient, substrateListener, nil, blockstore, config, sExecutor)

				chains = append(chains, substrateChain)
			}
		default:
			panic(fmt.Errorf("type '%s' not recognized", chainConfig["type"]))
		}
	}

	meter, err := opentelemetry.DefaultMeter(context.Background(), configuration.RelayerConfig.OpenTelemetryCollectorURL)
	if err != nil {
		panic(err)
	}
	metrics := metrics.NewTelemetry(meter, configuration.RelayerConfig.Env, configuration.RelayerConfig.Id)

	go jobs.StartCommunicationHealthCheckJob(host, configuration.RelayerConfig.MpcConfig.CommHealthCheckInterval, metrics)

	r := relayer.NewRelayer(
		chains,
		metrics,
	)

	errChn := make(chan error)
	go r.Start(ctx, errChn)

	sysErr := make(chan os.Signal, 1)
	signal.Notify(sysErr,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGHUP,
		syscall.SIGQUIT)

	relayerName := viper.GetString("name")
	log.Info().Msgf("Started relayer: %s with PID: %s", relayerName, host.ID().Pretty())

	_, err = keyshareStore.GetKeyshare()
	if err != nil {
		log.Info().Msg("Relayer not part of MPC. Waiting for refresh event...")
	}

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
