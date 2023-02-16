// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package app

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"syscall"

	"github.com/ChainSafe/chainbridge-core/lvldb"

	"github.com/ethereum/go-ethereum/common"
	secp256k1 "github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	coreEvm "github.com/ChainSafe/chainbridge-core/chains/evm"
	coreEvents "github.com/ChainSafe/chainbridge-core/chains/evm/calls/events"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmclient"
	"github.com/ChainSafe/chainbridge-core/relayer/message"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"

	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/evmtransaction"
	"github.com/ChainSafe/chainbridge-core/chains/evm/calls/transactor/signAndSend"
	coreExecutor "github.com/ChainSafe/chainbridge-core/chains/evm/executor"
	coreListener "github.com/ChainSafe/chainbridge-core/chains/evm/listener"
	"github.com/ChainSafe/chainbridge-core/e2e/dummy"
	"github.com/ChainSafe/chainbridge-core/flags"
	"github.com/ChainSafe/chainbridge-core/opentelemetry"
	"github.com/ChainSafe/chainbridge-core/relayer"
	"github.com/ChainSafe/chainbridge-core/store"

	"github.com/ChainSafe/sygma-relayer/chains/evm"
	"github.com/ChainSafe/sygma-relayer/chains/substrate"
	substrate_bridge "github.com/ChainSafe/sygma-relayer/chains/substrate/calls/pallets/bridge"
	"github.com/ChainSafe/sygma-relayer/chains/substrate/client"
	"github.com/ChainSafe/sygma-relayer/chains/substrate/connection"
	substrate_events "github.com/ChainSafe/sygma-relayer/chains/substrate/events"
	substrateExecutor "github.com/ChainSafe/sygma-relayer/chains/substrate/executor"
	substrate_listener "github.com/ChainSafe/sygma-relayer/chains/substrate/listener"

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
	var TestKeyringPairAlice = signature.KeyringPair{
		URI:       "//Alice",
		PublicKey: []byte{0xd4, 0x35, 0x93, 0xc7, 0x15, 0xfd, 0xd3, 0x1c, 0x61, 0x14, 0x1a, 0xbd, 0x4, 0xa9, 0x9f, 0xd6, 0x82, 0x2c, 0x85, 0x58, 0x85, 0x4c, 0xcd, 0xe3, 0x9a, 0x56, 0x84, 0xe7, 0xa5, 0x6d, 0xa2, 0x7d},
		Address:   "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
	}

	configuration, err := config.GetConfigFromFile(viper.GetString(flags.ConfigFlagName))
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

	chains := []relayer.RelayedChain{}
	for _, chainConfig := range configuration.ChainConfigs {
		switch chainConfig["type"] {
		case "evm":
			{
				config, err := evm.NewEVMConfig(chainConfig)
				if err != nil {
					panic(err)
				}

				privateKey, err := secp256k1.HexToECDSA(config.GeneralChainConfig.Key)
				if err != nil {
					panic(err)
				}

				client, err := evmclient.NewEVMClient(config.GeneralChainConfig.Endpoint, privateKey)
				if err != nil {
					panic(err)
				}

				mod := big.NewInt(0).Mod(config.StartBlock, config.BlockConfirmations)
				// startBlock % blockConfirmations == 0
				if mod.Cmp(big.NewInt(0)) != 0 {
					config.StartBlock.Sub(config.StartBlock, mod)
				}

				bridgeAddress := common.HexToAddress(config.Bridge)
				dummyGasPricer := dummy.NewStaticGasPriceDeterminant(client, nil)
				t := signAndSend.NewSignAndSendTransactor(evmtransaction.NewTransaction, dummyGasPricer, client)
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
				eventHandlers = append(eventHandlers, listener.NewRefreshEventHandler(l, nil, nil, tssListener, coordinator, host, communication, connectionGate, keyshareStore, bridgeAddress))
				eventHandlers = append(eventHandlers, listener.NewRetryEventHandler(l, tssListener, depositHandler, bridgeAddress, *config.GeneralChainConfig.Id, config.BlockConfirmations))
				evmListener := coreListener.NewEVMListener(client, eventHandlers, blockstore, *config.GeneralChainConfig.Id, config.BlockRetryInterval, config.BlockConfirmations, config.BlockInterval)
				executor := executor.NewExecutor(host, communication, coordinator, mh, bridgeContract, keyshareStore)

				coreEvmChain := coreEvm.NewEVMChain(evmListener, nil, blockstore, *config.GeneralChainConfig.Id, config.StartBlock, config.GeneralChainConfig.LatestBlock, config.GeneralChainConfig.FreshStart)
				chain := evm.NewEVMChain(*coreEvmChain, executor)

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

				client, err := client.NewSubstrateClient(config.GeneralChainConfig.Endpoint, &TestKeyringPairAlice, config.ChainID)
				if err != nil {
					panic(err)
				}
				mod := big.NewInt(0).Mod(config.StartBlock, config.BlockConfirmations)
				// startBlock % blockConfirmations == 0
				if mod.Cmp(big.NewInt(0)) != 0 {
					config.StartBlock.Sub(config.StartBlock, mod)
				}

				bridgePallet := substrate_bridge.NewBridgePallet(client)

				depositHandler := substrate_events.NewSubstrateDepositHandler()
				depositHandler.RegisterDepositHandler(message.FungibleTransfer, substrate_events.FungibleTransferHandler)
				eventHandlers := make([]substrate_listener.EventHandler, 0)
				eventHandlers = append(eventHandlers, substrate_events.NewFungibleTransferEventHandler(*config.GeneralChainConfig.Id, depositHandler))
				substrateListener := substrate_listener.NewSubstrateListener(conn, eventHandlers, config)

				mh := substrateExecutor.NewSubstrateMessageHandler()
				mh.RegisterMessageHandler(message.FungibleTransfer, substrateExecutor.FungibleTransferMessageHandler)

				executor := substrateExecutor.NewExecutor(host, communication, coordinator, mh, bridgePallet, keyshareStore, conn)

				substrateChain := substrate.NewSubstrateChain(substrateListener, nil, blockstore, config, executor)

				chains = append(chains, substrateChain)

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

	select {
	case err := <-errChn:
		log.Error().Err(err).Msg("failed to listen and serve")
		return err
	case sig := <-sysErr:
		log.Info().Msgf("terminating got ` [%v] signal", sig)
		return nil
	}
}
