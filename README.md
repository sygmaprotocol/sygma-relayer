# Chainbridge Hub

<a href="https://golang.org">
<img alt="go" src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" />
</a>

&nbsp;

:construction: *This is still unstable MVP implementation of ChainBridge HUB* :construction:

ChainBridge Hub uses [chainbridge-core](https://github.com/ChainSafe/chainbridge-core) framework and replaces relayer voting mechanism with MPC signing process.

&nbsp;

### Table of Contents

1. [Local environment](#local-environment)
2. [Configuration](#configuration)

&nbsp;

## Local environment
Run `make example` to start local environment.

_This will start 2 evm networks, deploy and configure smart contracts, and start 3 preconfigured relayers._

&nbsp;

## Configuration

Configuration can be provided either as configuration file or as ENV variables, depending on `--config` flag value.
If it is set to `env` configuration properties are expected as ENV variables, otherwise you need to provide path to configuration file.

Configuration consists of two distinct parts. First one being `RelayerConfig` defining various parameters of relayer. 
Second one is list of `ChainConfig` that define parameters for each chain/domain that relayer is processing.

### Configuration file

Example of json configuration file can be found inside `/example/cfg` folder.

### ENV variables 

_Each ENV variable needs to be prefixed with CBH._

Properties of `RelayerConfig` are expected to be defined as separate ENV variables
where ENV variable name reflects properties position in structure.

For example, if you want to set `Config.RelayerConfig.MpcConfig.Port` this would
translate to ENV variable named `CBH_RELAYER_MPCCONFIG_PORT`.

Each `ChainConfig` is defined as one ENV variable, where its content is JSON configuration for one chain/domain.
Variables are named like this: `CBH_DOM_X` where `X` is domain id.