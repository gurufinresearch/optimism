# Custom Deployments

While OP Deployer was designed primarily for use with chains that are governed by Optimism, it also supports managing
custom deployments. This is particularly common for RaaS providers, whose customers often request deployments with
custom L1s (or L2s, in the case of L3s) or governance. This guide will walk you through the process of managing these
chains using OP Deployer.

> [!danger]
> Chains deployed in this way are not subject to Optimism Governance. They may be running customized or unaudited
> code. Use at your own risk.

## Bootstrapping

The first step to deploying a custom Superchain is to bootstrap it onto an L1. This process will:

- Deploy Superchain-wide management contracts like `SuperchainConfig` and `SuperchainVersions`.
- Deploy contract implementations that will be shared among all OP Chains deployed using this Superchain.
- Set up ownership so that you can control the Superchain.

You will use the `bootstrap` [family][bootstrap] of commands on `op-deployer` to do this.

[bootstrap]: ../user-guide/bootstrap.md

### Superchain Bootstrap

Every OP Chain belongs to a logical Superchain. The Superchain consists of a set of contracts that control the behavior
of a group of OP Chains. This includes:

- Pausing bridges
- Signaling which protocol versions are required and recommended

You can deploy a new Superchain for each OP Chain, or you can deploy it once and share it across multiple OP Chains.
The choice is up to the deployer. Note that you **cannot** share a Superchain across multiple L1s.

To begin, bootstrap the Superchain onto your chosen L1 with the following command:

```shell
op-deployer bootstrap superchain \
  --l1-rpc-url="<rpc url>" \
  --private-key="<contract deployer private key>" \
  --artifacts-locator="<locator>" \
  --outfile="<path to outfile>" \
  --superchain-proxy-admin-owner="<role address>" \
  --protocol-versions-owner="<role address>" \
  --guardian="<role address>"
```

This will output a JSON file containing the addresses of the relevant contracts. Keep track of this file, as you will
need it in subsequent steps.

We recommend the following these best practices when bootstrapping the Superchain:

1. Use Gnosis SAFEs for ownership roles like `guardian` and `superchain-proxy-admin-owner`. The owner **must** be a
   smart contract to support future upgrades, so a SAFE is a sensible default.
2. Use a regular EOA as the deployer. It will not have any control over the Superchain once the deployment completes.
3. Use a standard contracts tag (e.g. `tag://op-contracts/v2.0.0`). This will make upgrading easier.

### Bootstrapping Implementations

The smart contracts for an OP Chain are deployed using a factory that points to a set of predeployed implementations.
You must deploy the factory and the implementations every time you deploy to a new L1 and whenever new smart
contract versions are released. Implementations are deployed using `CREATE2`, so addresses may be reused if they
already exist on the L1.

> [!warning]
> You may need to use different versions of OP Deployer depending on which contracts version you are deploying. See the
> [releases guide][releases] for more information on picking the right release.

[releases]: ./releases.md

To deploy the implementations, use the following command:

```shell
op-deployer bootstrap implementations \
  --artifacts-locator="<locator, should be the same as the one used in bootstrap superchain>" \
  --l1-rpc-url="<rpc url>" \
  --outfile="<path to outfile>" \
  --mips-version="2" \
  --private-key="<contract deployer private key>" \
  --protocol-versions-proxy="<address output from bootstrap superchain>" \
  --superchain-config-proxy="<address output from bootstrap superchain>" \
  --upgrade-controller="<superchain-proxy-admin-owner used in bootstrap superchain>"
```

Similar to the Superchain bootstrap command, this will output a JSON file containing the addresses of the relevant
contracts. Again, keep track of this file.

The most important address in the implementations file is the OPCM, or OP Contracts Manager. This contract is the
factory that will deploy all the OP Chains belonging to this Superchain. It is also responsible for upgrading between
different contracts versions. Please keep the following **very important** invariants in mind with the OPCM:

- There is a one-to-one mapping between each OPCM, and contracts version.
- Each OPCM is associated with **exactly one** Superchain. This means that you **must** deploy a new OPCM using the
  `bootstrap implementations` command for each new Superchain you manage.

## Deploying

After bootstrapping the Superchain and implementations, you can deploy your L2 chains with the `apply` command. You
will need to specify a `configType` of `standard-overrides` and set the `opcmAddress` field in your intent to the
address of the OPCM above. **Make sure you call the right OPCM.** Failing to call the right OPCM might lead to
deploying incorrect contract versions, or associating your chain with the wrong Superchain.

> [!warning]
> Make sure that you use the same `l1ContractsLocator` and `l2ContractsLocator` as the ones used in the bootstrap
> commands. Otherwise, you may run into deployment errors.

See the following config for an example:

```toml
configType = "standard-overrides"
l1ChainID = 11155420
opcmAddress = "0x..."
l1ContractsLocator = "tag://..." # must match the one used in bootstrap
l2ContractsLocator = "tag://..."

[[chains]]
# Chain configs...
```

Once `apply` completes successfully, you can use the `inspect` family of commands to download your chain's L2
genesis and rollup config files.

### Custom L1 chains

Out of the box, the `standard` and `standard-overrides` intent types only know about Ethereum mainnet (chain ID `1`)
and Sepolia (chain ID `11155111`). Those are the only chains in the superchain registry, which is where OP Deployer
reads the OPCM address and the standard role addresses from. Running `op-deployer init --intent-type
standard-overrides` against any other L1 fails with `unsupported chain ID`.

To deploy onto a custom L1, describe it with the `CUSTOM_CHAIN_*` environment variables. When `CUSTOM_CHAIN_ID` is set
to a non-zero value, OP Deployer registers an additional L1 alongside mainnet and Sepolia for the duration of the
process:

| Variable | Description |
| --- | --- |
| `CUSTOM_CHAIN_ID` | EIP-155 chain ID of the L1. Required. Must not be `1` or `11155111`. |
| `CUSTOM_CHAIN_NAME` | Human readable name used in error messages. Defaults to `custom-<id>`. |
| `CUSTOM_CHAIN_OPCM_IMPL` | OPCM implementation address, taken from the `bootstrap implementations` outfile. |
| `CUSTOM_CHAIN_OPCM_TAG` | Contracts tag the OPCM address belongs to. Defaults to the current tag. |
| `CUSTOM_CHAIN_GUARDIAN` | Guardian role address. |
| `CUSTOM_CHAIN_CHALLENGER` | Challenger role address. |
| `CUSTOM_CHAIN_L1_PROXY_ADMIN_OWNER` | L1 ProxyAdmin owner role address. |
| `CUSTOM_CHAIN_L2_PROXY_ADMIN_OWNER` | L2 ProxyAdmin owner role address. |
| `CUSTOM_CHAIN_PROTOCOL_VERSIONS_OWNER` | ProtocolVersions owner role address. |
| `CUSTOM_CHAIN_SUPERCHAIN_PROXY_ADMIN` | Superchain ProxyAdmin address. |
| `CUSTOM_CHAIN_EXPLORER_API_URL` | Etherscan-compatible API endpoint (for example a Blockscout `/api`), used by `verify`. |

These values seed the generated `intent.toml`, which remains the source of truth once `init` has run. Roles you leave
unset come through as zero addresses and are rejected by `apply`, exactly as they are with the `custom` intent type.

```shell
export CUSTOM_CHAIN_ID=4242
export CUSTOM_CHAIN_NAME=my-l1
export CUSTOM_CHAIN_OPCM_IMPL=0x...   # from the bootstrap implementations outfile
export CUSTOM_CHAIN_CHALLENGER=0x...
export CUSTOM_CHAIN_L1_PROXY_ADMIN_OWNER=0x...
export CUSTOM_CHAIN_L2_PROXY_ADMIN_OWNER=0x...

op-deployer init \
  --intent-type standard-overrides \
  --l1-chain-id "$CUSTOM_CHAIN_ID" \
  --l2-chain-ids 111 \
  --workdir .deployer
```

A custom L1 is not part of a registered Superchain, so commands that read superchain registry data for the L1 report
`is not in the superchain registry` rather than silently falling back to mainnet's configuration.

The `custom` intent type remains available and needs none of this. It leaves every field for you to fill in by hand
and works with any L1 chain ID.

## Upgrading

You can upgrade between smart contract versions using the `upgrade` [family of commands][upgrade]. As a prerequisite,
you must deploy OPCMs for each version you want to upgrade between using the `bootstrap implementations` command.

[upgrade]: ../user-guide/upgrade.md

You will need to use the correct sub-command for the version you are upgrading from. For example, if you are
upgrading from `v2.0.0` to `v3.0.0`, you will need to use the `upgrade v3.0.0` subcommand.

Unlike the `bootstrap` and `apply` commands, the `upgrade` command does not interact with the chain directly. Instead,
it generates calldata that you can use with `cast`, Gnosis SAFE, or whatever tooling you use to manage your L1.

To run the upgrade command, use the following:

> [!warning]
> Upgrading between non-standard contract versions is not supported.

```shell
op-deployer upgrade <version> \
  --config <path to config JSON> \
  --l1-rpc-url="<rpc url>"
```

The contents of your config JSON should look something like this:

```json
{
  "prank": "<address of the contract that owns the chain - likely the same as the superchchain owner>",
  "opcm": "<address of new OPCM>",
  "chainConfigs": [
    {
      "systemConfigProxy": "<address of the chain's system config proxy>",
      "proxyAdmin": "<address of the chain's proxy admin>",
      "absolutePrestate": "<32-byte hash of the chain's absolute prestate>"
    }
  ]
}
```

You will get output that looks something like this:

```json
{
  "to": "<maps to the prank address>",
  "data": "<calldata>",
  "value": "0x0"
}
```

At this point, you will need to build a transaction that uses the calldata and calls the `upgrade()` method on the
OPCM. The exact method you use to do this will depend on your tooling. As an example, you can craft a transaction
for use with Gnosis SAFE CLI using the command below:

> [!info]
> The Gnosis SAFE UI does not support the `--delegate` flag, so the CLI is required if you're using a Gnosis SAFE.

```shell
safe-cli send-custom <owner SAFE address> <l1 rpc URL> <opcm address> 0 <calldata> --private-key <signer private key> 
--delegate
```

Note that no matter which method you use to broadcast the calldata, the call to OPCM **must** come from the smart
contract that owns the chain and the call must be via `DELEGATECALL`. If your upgrade command reverts, it is likely
due to one of these conditions not being met.