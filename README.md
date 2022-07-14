# Sygma bridge

<a href="https://golang.org">
<img alt="go" src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" />
</a>

&nbsp;

:construction: *This is still unstable MVP implementation of ChainBridge HUB* :construction:

ChainBridge Hub uses [chainbridge-core](https://github.com/ChainSafe/chainbridge-core) framework and replaces the relayer voting mechanism with the MPC signing process.

&nbsp;

### Table of Contents

1. [Local environment](#local-environment)
2. [Configuration](#configuration)

&nbsp;

## Local environment
Run `make example` to start the local environment.

_This will start two EVM networks, deploy and configure smart contracts, and start three preconfigured relayers._

&nbsp;

## Configuration

Configuration can be provided either as a configuration file or as ENV variables, depending on `--config` flag value.
If it is set to `env`, configuration properties are expected as ENV variables. Otherwise, you need to provide a path to the configuration file.

Configuration consists of two distinct parts. The first one is `RelayerConfig`, defining various parameters of the relayer.
The second one is a list of `ChainConfig` that defines parameters for each chain/domain that the relayer is processing.

### Configuration file

An example of a JSON configuration file can be found inside `/example/cfg` folder.

### ENV variables

_Each ENV variable needs to be prefixed with CBH._

Properties of `RelayerConfig` are expected to be defined as separate ENV variables
where ENV variable name reflects properties position in the structure.

For example, if you want to set `Config.RelayerConfig.MpcConfig.Port` this would
translate to ENV variable named `CBH_RELAYER_MPCCONFIG_PORT`.

Each `ChainConfig` is defined as one ENV variable, where its content is JSON configuration for one chain/domain.
Variables are named like this: `CBH_DOM_X` where `X` is domain id.
