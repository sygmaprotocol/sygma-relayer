// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ChainSafe/chainbridge-core/crypto/secp256k1"
	"github.com/ChainSafe/chainbridge-core/lvldb"
	"github.com/ChainSafe/chainbridge-core/opentelemetry"

	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	coreEvents "github.com/ChainSafe/chainbridge-core/chains/evm/calls/events"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmclient"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor/monitored"
	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmtransaction"
	coreExecutor "github.com/ChainSafe/chainbridge-core/chains/evm/executor"
	coreListener "github.com/ChainSafe/chainbridge-core/chains/evm/listener"
	"github.com/ChainSafe/chainbridge-core/e2e/dummy"
	"github.com/ChainSafe/chainbridge-core/flags"
	"github.com/ChainSafe/chainbridge-core/relayer"
	"github.com/ChainSafe/chainbridge-core/store"

	"github.com/ChainSafe/sygma-relayer/chains/evm"
	"github.com/ChainSafe/sygma-relayer/chains/substrate"
	"github.com/ChainSafe/sygma-relayer/chains/substrate/client"
	"github.com/ChainSafe/sygma-relayer/chains/substrate/connection"
	substrateExecutor "github.com/ChainSafe/sygma-relayer/chains/substrate/executor"
	substrate_listener "github.com/ChainSafe/sygma-relayer/chains/substrate/listener"
	substrate_pallet "github.com/ChainSafe/sygma-relayer/chains/substrate/pallet"
	"github.com/ChainSafe/sygma-relayer/jobs"
	"github.com/ChainSafe/sygma-relayer/metrics"

	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/contracts/bridge"
	"github.com/ChainSafe/sygma-relayer/chains/evm/calls/events"
	"github.com/ChainSafe/sygma-relayer/chains/evm/executor"
	"github.com/ChainSafe/sygma-relayer/chains/evm/listener"
	"github.com/ChainSafe/sygma-relayer/comm/elector"
	"github.com/ChainSafe/sygma-relayer/comm/p2p"
	"github.com/ChainSafe/sygma-relayer/config"
	"github.com/ChainSafe/sygma-relayer/keyshare"
	"github.com/ChainSafe/sygma-relayer/topology"
	"github.com/ChainSafe/sygma-relayer/tss"
)

func Run() error {
	configuration := &config.Config{}
	configuration, err := config.GetConfigFromFile(viper.GetString(flags.ConfigFlagName), configuration)
	if err != nil {
		panic(err)
	}

	networkTopology, _ := topology.ProcessRawTopology(&topology.RawTopology{
		Peers: []topology.RawPeer{
			{PeerAddress: "/dns4/relayer2/tcp/9001/p2p/QmeTuMtdpPB7zKDgmobEwSvxodrf5aFVSmBXX3SQJVjJaT"},
			{PeerAddress: "/dns4/relayer3/tcp/9002/p2p/QmYAYuLUPNwYEBYJaKHcE7NKjUhiUV8txx2xDXHvcYa1xK"},
			{PeerAddress: "/dns4/relayer1/tcp/9000/p2p/QmcvEg7jGvuxdsUFRUiE4VdrL2P1Yeju5L83BsJvvXz7zX"},
		},
		Threshold: "2",
	})

	db, err := lvldb.NewLvlDB(viper.GetString(flags.BlockstoreFlagName))
	if err != nil {
		panic(err)
	}
	blockstore := store.NewBlockStore(db)

	privBytes, err := crypto.ConfigDecodeKey(configuration.RelayerConfig.MpcConfig.Key)
	if err != nil {
		panic(err)
	}
	priv, err := crypto.UnmarshalPrivateKey(privBytes)
	if err != nil {
		panic(err)
	}

	connectionGate := p2p.NewConnectionGate(networkTopology)
	host, err := p2p.NewHost(priv, networkTopology, connectionGate, configuration.RelayerConfig.MpcConfig.Port)
	if err != nil {
		panic(err)
	}

	communication := p2p.NewCommunication(host, "p2p/sygma")
	electorFactory := elector.NewCoordinatorElectorFactory(host, configuration.RelayerConfig.BullyConfig)
	coordinator := tss.NewCoordinator(host, communication, electorFactory)
	keyshareStore := keyshare.NewKeyshareStore(configuration.RelayerConfig.MpcConfig.KeysharePath)

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
				if err != nil {
					panic(err)
				}

				kp, err := secp256k1.NewKeypairFromString(config.GeneralChainConfig.Key)
				if err != nil {
					panic(err)
				}

				client, err := evmclient.NewEVMClient(config.GeneralChainConfig.Endpoint, kp)
				if err != nil {
					panic(err)
				}

				log.Info().Str("domain", config.String()).Msgf("Registering EVM domain")

				bridgeAddress := common.HexToAddress(config.Bridge)
				dummyGasPricer := dummy.NewStaticGasPriceDeterminant(client, nil)
				t := monitored.NewMonitoredTransactor(evmtransaction.NewTransaction, dummyGasPricer, client, config.MaxGasPrice, config.GasIncreasePercentage)
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
				l := log.With().Str("chain", fmt.Sprintf("%v", chainConfig["name"])).Uint8("domainID", *config.GeneralChainConfig.Id)
				eventHandlers = append(eventHandlers, listener.NewDepositEventHandler(l, depositListener, depositHandler, bridgeAddress, *config.GeneralChainConfig.Id))
				eventHandlers = append(eventHandlers, listener.NewKeygenEventHandler(l, tssListener, coordinator, host, communication, keyshareStore, bridgeAddress, networkTopology.Threshold))
				eventHandlers = append(eventHandlers, listener.NewRefreshEventHandler(l, nil, nil, tssListener, coordinator, host, communication, connectionGate, keyshareStore, bridgeAddress))
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

				log.Info().Str("domain", config.String()).Msgf("Registering substrate domain")

				client := client.NewSubstrateClient(conn, &keyPair, config.ChainID, config.Tip)
				bridgePallet := substrate_pallet.NewPallet(client)

				l := log.With().Str("chain", fmt.Sprintf("%v", chainConfig["name"])).Uint8("domainID", *config.GeneralChainConfig.Id)
				depositHandler := substrate_listener.NewSubstrateDepositHandler()
				depositHandler.RegisterDepositHandler(message.FungibleTransfer, substrate_listener.FungibleTransferHandler)
				eventHandlers := make([]substrate_listener.EventHandler, 0)
				eventHandlers = append(eventHandlers, substrate_listener.NewFungibleTransferEventHandler(l, *config.GeneralChainConfig.Id, depositHandler))
				eventHandlers = append(eventHandlers, substrate_listener.NewRetryEventHandler(l, conn, depositHandler, *config.GeneralChainConfig.Id))
				substrateListener := substrate_listener.NewSubstrateListener(conn, eventHandlers, config)

				mh := substrateExecutor.NewSubstrateMessageHandler()
				mh.RegisterMessageHandler(message.FungibleTransfer, substrateExecutor.FungibleTransferMessageHandler)

				executor := substrateExecutor.NewExecutor(host, communication, coordinator, mh, bridgePallet, keyshareStore, conn, exitLock)

				substrateChain := substrate.NewSubstrateChain(substrateListener, nil, blockstore, config, executor)

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

	select {
	case err := <-errChn:
		log.Error().Err(err).Msg("failed to listen and serve")
		return err
	case sig := <-sysErr:
		log.Info().Msgf("terminating got ` [%v] signal", sig)
		return nil
	}
}
