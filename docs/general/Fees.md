# Fees

Sygma allows granular control of handling fees for each resource. Even though specific implementations differ based on chain architecture, general functionality is the same. The concept is that each resource is assigned a fee strategy for every potential destination domain, with this mapping also outlining all potential bridging routes for a given resource.

![](https://i.imgur.com/7fG5XLO.png)

## Fee strategies

Fee strategy defines a set of rules on how fees should be charged when executing deposits on the source chain.

### Static fee strategy

This strategy always requires a predefined static fee amount per deposit. **It can only collect fees in the native currency of the source chain**.

*On the diagram below, we use Sygma SDK for interaction with all services.*

![](https://i.imgur.com/BW3j9IL.png)

#### Deposit flow

1) Calculate the final fee
    - Based on resourceID and domainsID, request a final fee amount that will be required to execute the deposit.
2) Execute deposit
    - Send the appropriate base currency amount based on the calculated final fee when executing the deposit.

### Dynamic fee strategy
This strategy utilizes the [Sygma Fee Oracle service](https://github.com/sygmaprotocol/sygma-fee-oracle/wiki), which issues fee estimates with details on the gas price for the destination chain. In addition, fee oracle can provide price information for different tokens, enabling paying bridging fees in the not native currency. Each issued gas estimate has a limited time validity in which it needs to be executed.

Check out the [Sygma Fee Oracle service repository](https://github.com/sygmaprotocol/sygma-fee-oracle/wiki) for more details on the service and the format of the issued fee estimates.

*On the diagram below, we use Sygma SDK for interaction with all services.*

![](https://i.imgur.com/3ABwdf2.png)

#### Deposit flow

1) Fetch fee estimate
    - Based on resource ID and domains ID, request fee estimate from Fee Oracle service. This fee estimate is valid until `expiresAt`.
2) Validate fee estimate and calculate the final fee
    - Validate the signature on the fee estimate.
    - Get the final fee amount that will be collected on deposit.
3) Execute deposit
    - Provide fee estimate data as an argument when executing the deposit.

## Implementations

#### EVM
Check out the [solidity documentation](https://github.com/sygmaprotocol/sygma-solidity/wiki/Fees) for details on EVM implementation.

#### Substrate
WIP


 