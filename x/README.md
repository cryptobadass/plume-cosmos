<!--
parent:
  order: false
-->

# List of Modules

Here are some production-grade modules that can be used in Cosmos SDK applications, along with their respective documentation:

- [Accesscontrol] - Resource dependency access control module used for managing concurrent read/write access to resources.
- [Auth](auth/spec/README.md) - Authentication of accounts and transactions for Cosmos SDK application.
- [Authz](authz/spec/README.md) - Authorization for accounts to perform actions on behalf of other accounts.
- [Bank](bank/spec/README.md) - Token transfer functionalities.
- [Capability](capability/spec/README.md) - Object capability implementation.
- [Crisis](crisis/spec/README.md) - Halting the blockchain under certain circumstances (e.g. if an invariant is broken).
- [Distribution](distribution/spec/README.md) - Fee distribution, and staking token provision distribution.
- [Evidence](evidence/spec/README.md) - Evidence handling for double signing, misbehaviour, etc.
- [Governance](gov/spec/README.md) - On-chain proposals and voting.
- [Mint](mint/spec/README.md) - Creation of new units of staking token.
- [Params](params/spec/README.md) - Globally available parameter store.
- [Slashing](slashing/spec/README.md) - Validator punishment mechanisms.
- [Staking](staking/spec/README.md) - Proof-of-Stake layer for public blockchains.
- [Upgrade](upgrade/spec/README.md) - Software upgrades handling and coordination.

To learn more about the process of building modules, visit the [building modules reference documentation](../docs/building-modules/README.md).

## IBC

The IBC module for the SDK has moved to its [own repository](https://github.com/cosmos/ibc-go).

### FeesParams

To query for current fee params:

```bash
plume q params feesparams 
```

To update the feesparams, use a governance proposal like such:

```json
{
  "title": "Update Global Minimum Prices",
  "description": "This proposal seeks update the global minimum prices for a gas unit.",
  "changes": [
    {
      "subspace": "params",
      "key": "FeesParams",
      "value": {
	  "global_minimum_gas_prices": [
    		{
      		"denom": "uplume",
      		"amount":	 "1.00000000000000000"
    		}
  	]
 	}
    }
  ],
  "deposit": "1000000000uplume",
  "is_expedited": true
}
```
