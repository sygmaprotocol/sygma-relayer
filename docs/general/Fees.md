# Fees

Sygma allows granular control of handling fees for each resource. Even though specific implementations differ based on
chain architecture, general functionality is the same. The concept is that each resource is assigned a fee strategy for
every potential destination domain, with this mapping also outlining all potential bridging routes for a given resource.

![](/docs/resources/fee-router-general.png)

## Fee strategies

Fee strategy defines a set of rules on how fees should be charged when executing deposits on the source chain.

### Fixed fee strategy

This strategy always requires a predefined fixed fee amount per deposit. **It can only collect fees in the native
currency of the source chain**.

*On the diagram below, we use [Sygma SDK](https://github.com/sygmaprotocol/sygma-sdk) for interaction with all services.*

![](/docs/resources/static-fee-general.png)

#### Deposit flow

1) Calculate the final fee
    - Based on resourceID and domainsID, request a final fee amount that will be required to execute the deposit.
2) Execute deposit
    - Send the appropriate base currency amount based on the calculated final fee when executing the deposit.

### Percentage based fee strategy

This strategy calculates fee amount based on the amount of token being transferred. 
It always collects fee in token that is being transferred, so it only makes sense for fungible token routes.

<img src="/docs/resources/percentage-formula-general.png" data-canonical-src="/docs/resources/percentage-formula-general.png" width="386" height="267" />

*On the diagram below, we use [Sygma SDK](https://github.com/sygmaprotocol/sygma-sdk) for interaction with all services.*

![](/docs/resources/percentage-fee-general.png)

#### Deposit flow

1) Calculate the final fee
   - Based on resourceID, domainsID and amount, request a final fee amount that will be required to execute the deposit.
2) Execute deposit
   - Send the appropriate token amount based on the calculated final fee when executing the deposit.

### Dynamic fee strategy (EVM only)

This strategy utilizes the [Sygma Fee Oracle service](https://github.com/sygmaprotocol/sygma-fee-oracle), which issues
fee estimates with details on the gas price for the destination chain. In addition, fee oracle can provide price
information for different tokens, enabling paying bridging fees in the not native currency. Each issued gas estimate has
a limited time validity in which it needs to be executed.

Check out
the [Sygma Fee Oracle technical documentation](https://github.com/sygmaprotocol/sygma-fee-oracle/blob/main/docs/Home.md) for
more details on the service and the format of the issued fee estimates.

*On the diagram below, we use [Sygma SDK](https://github.com/sygmaprotocol/sygma-sdk) for interaction with all services.*

![](/docs/resources/dynamic-fee-general.png)

#### Deposit flow

1) Fetch fee estimate
    - Based on resource ID and domains ID, request fee estimate from Fee Oracle service. This fee estimate is valid
      until `expiresAt`.
2) Validate fee estimate and calculate the final fee
    - Validate the signature on the fee estimate.
    - Get the final fee amount that will be collected on deposit.
3) Execute deposit
    - Provide fee estimate data as an argument when executing the deposit.

## Implementations

#### EVM

Check out the [solidity documentation](https://github.com/sygmaprotocol/sygma-solidity/blob/main/docs/Home.md) for
details on EVM implementation.

#### Substrate

Check out
the [substrate pallet documentation](https://github.com/sygmaprotocol/sygma-substrate-pallets/blob/main/docs/Home.md)
for details on Substrate implementation.


 