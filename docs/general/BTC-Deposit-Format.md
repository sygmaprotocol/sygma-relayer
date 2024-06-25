# BTC Deposit Transaction Format

This document describes the expected format for BTC deposit transactions and how the deposit amount is calculated.

## Expected BTC Deposit Transaction Format

### OP_RETURN Output

- **Purpose**: Stores arbitrary data within the transaction.
- **Requirements**:
  - There should be at most one output with a `ScriptPubKey.Type` of `OP_RETURN`.
  - The `OP_RETURN` data must be formatted as `receiverEVMAddress_destinationDomainID`.


## Amount Calculation

- The total deposit amount is calculated by summing the values of the outputs that match the resource address.
- Only outputs with script types of `WitnessV0KeyHash` are considered for the amount calculation.
