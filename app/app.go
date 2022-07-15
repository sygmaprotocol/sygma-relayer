// Copyright 2021 ChainSafe Systems
// SPDX-License-Identifier: LGPL-3.0-only

package app

import (
	"context"
	"fmt"
	coreEvm "github.com/ChainSafe/chainbridge-core/chains/evm"
	coreEvents "github.com/ChainSafe/chainbridge-core/chains/evm/calls/events"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmclient"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmgaspricer"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmtransaction"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor/signAndSend"
	coreExecutor "github.com/ChainSafe/chainbridge-core/chains/evm/executor"
	coreListener "github.com/ChainSafe/chainbridge-core/chains/evm/listener"
	"github.com/ChainSafe/chainbridge-core/config/chain"
	"github.com/ChainSafe/chainbridge-core/logger"
	"github.com/ChainSafe/chainbridge-core/lvldb"
	"github.com/ChainSafe/chainbridge-core/opentelemetry"
	"github.com/ChainSafe/chainbridge-core/relayer"
	"github.com/ChainSafe/chainbridge-core/store"
	"github.com/ChainSafe/chainbridge-hub/chains/evm"
	"github.com/ChainSafe/chainbridge-hub/chains/evm/calls/contracts/bridge"
	"github.com/ChainSafe/chainbridge-hub/chains/evm/calls/events"
	"github.com/ChainSafe/chainbridge-hub/chains/evm/executor"
	"github.com/ChainSafe/chainbridge-hub/chains/evm/listener"
	"github.com/ChainSafe/chainbridge-hub/comm/elector"
	"github.com/ChainSafe/chainbridge-hub/comm/p2p"
	"github.com/ChainSafe/chainbridge-hub/health"
	"github.com/ChainSafe/chainbridge-hub/keyshare"
	"github.com/ChainSafe/chainbridge-hub/topology"
	"github.com/ChainSafe/chainbridge-hub/tss"
	"github.com/ethereum/go-ethereum/common"
	secp256k1 "github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"math/big"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ChainSafe/chainbridge-core/flags"
	"github.com/ChainSafe/chainbridge-hub/config"
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

	// topologyProvider, err := topology.NewNetworkTopologyProvider(configuration.RelayerConfig.MpcConfig.TopologyConfiguration)
	topologyProvider, err := topology.NewFixedNetworkTopologyProvider()
	panicOnError(err)

	networkTopology, err := topologyProvider.NetworkTopology()
	panicOnError(err)

	var allowedPeers peer.IDSlice
	for _, pAdrInfo := range networkTopology.Peers {
		allowedPeers = append(allowedPeers, pAdrInfo.ID)
	}

	db, err := lvldb.NewLvlDB(viper.GetString(flags.BlockstoreFlagName))
	panicOnError(err)

	blockstore := store.NewBlockStore(db)

	privBytes, err := crypto.ConfigDecodeKey(configuration.RelayerConfig.MpcConfig.Key)
	panicOnError(err)

	priv, err := crypto.UnmarshalPrivateKey(privBytes)
	panicOnError(err)

	host, err := p2p.NewHost(priv, networkTopology, configuration.RelayerConfig.MpcConfig.Port)
	panicOnError(err)

	comm := p2p.NewCommunication(host, "p2p/chainbridge", allowedPeers)
	electorFactory := elector.NewCoordinatorElectorFactory(host, configuration.RelayerConfig.BullyConfig)
	coordinator := tss.NewCoordinator(host, comm, electorFactory)
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

				depositHandler := coreListener.NewETHDepositHandler(bridgeContract)
				depositHandler.RegisterDepositHandler(config.Erc20Handler, coreListener.Erc20DepositHandler)
				depositHandler.RegisterDepositHandler(config.Erc721Handler, coreListener.Erc721DepositHandler)
				depositHandler.RegisterDepositHandler(config.GenericHandler, coreListener.GenericDepositHandler)
				depositListener := coreEvents.NewListener(client)
				tssListener := events.NewListener(client)
				eventHandlers := make([]coreListener.EventHandler, 0)
				eventHandlers = append(eventHandlers, coreListener.NewDepositEventHandler(depositListener, depositHandler, bridgeAddress, *config.GeneralChainConfig.Id))
				eventHandlers = append(eventHandlers, listener.NewKeygenEventHandler(tssListener, coordinator, host, comm, keyshareStore, bridgeAddress, configuration.RelayerConfig.MpcConfig.Threshold))
				eventHandlers = append(eventHandlers, listener.NewRefreshEventHandler(topologyProvider, tssListener, coordinator, host, comm, keyshareStore, bridgeAddress, configuration.RelayerConfig.MpcConfig.Threshold))
				eventHandlers = append(eventHandlers, listener.NewRetryEventHandler(client, tssListener, depositHandler, bridgeAddress, *config.GeneralChainConfig.Id))
				evmListener := coreListener.NewEVMListener(client, eventHandlers, blockstore, config)

				mh := coreExecutor.NewEVMMessageHandler(bridgeContract)
				mh.RegisterMessageHandler(config.Erc20Handler, coreExecutor.ERC20MessageHandler)
				mh.RegisterMessageHandler(config.Erc721Handler, coreExecutor.ERC721MessageHandler)
				mh.RegisterMessageHandler(config.GenericHandler, coreExecutor.GenericMessageHandler)
				executor := executor.NewExecutor(host, comm, coordinator, mh, bridgeContract, keyshareStore)

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

	go health.StartHealthEndpoint(configuration.RelayerConfig.HealthPort)

	sysErr := make(chan os.Signal, 1)
	signal.Notify(sysErr,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGHUP,
		syscall.SIGQUIT)

	relayerName := viper.GetString("name")
	// log.Info().Msgf("Started relayer: %s with PID: %s", relayerName, host.ID().Pretty())
	log.Info().Msg(relayerName)
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
