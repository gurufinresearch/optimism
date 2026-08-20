package standard

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum-optimism/superchain-registry/validation"

	"github.com/ethereum/go-ethereum/common"
)

// Environment variables describing a custom L1 chain. Setting CustomChainIDEnvVar to a non-zero
// value registers an additional L1 profile, which makes the standard and standard-overrides
// intent types usable against an L1 that is not in the superchain registry.
const (
	CustomChainIDEnvVar                 = "CUSTOM_CHAIN_ID"
	CustomChainNameEnvVar               = "CUSTOM_CHAIN_NAME"
	CustomChainOPCMImplEnvVar           = "CUSTOM_CHAIN_OPCM_IMPL"
	CustomChainOPCMTagEnvVar            = "CUSTOM_CHAIN_OPCM_TAG"
	CustomChainGuardianEnvVar           = "CUSTOM_CHAIN_GUARDIAN"
	CustomChainChallengerEnvVar         = "CUSTOM_CHAIN_CHALLENGER"
	CustomChainL1ProxyAdminOwnerEnvVar  = "CUSTOM_CHAIN_L1_PROXY_ADMIN_OWNER"
	CustomChainL2ProxyAdminOwnerEnvVar  = "CUSTOM_CHAIN_L2_PROXY_ADMIN_OWNER"
	CustomChainProtocolVersionsOwnerVar = "CUSTOM_CHAIN_PROTOCOL_VERSIONS_OWNER"
	CustomChainSuperchainProxyAdminVar  = "CUSTOM_CHAIN_SUPERCHAIN_PROXY_ADMIN"
	CustomChainExplorerAPIURLEnvVar     = "CUSTOM_CHAIN_EXPLORER_API_URL"
)

// ErrUnsupportedChainID is returned when no L1 profile is registered for a chain ID. It is
// distinct from a malformed CUSTOM_CHAIN_* configuration, which must never be reported as an
// unsupported chain.
var ErrUnsupportedChainID = errors.New("unsupported chain ID")

// L1Profile holds everything OP Deployer needs to know about an L1 chain in order to populate
// and validate a standard or standard-overrides intent.
type L1Profile struct {
	// Name is a human readable name for the chain, used in error messages.
	Name string
	// ChainID is the EIP-155 chain ID of the L1.
	ChainID uint64
	// SuperchainName is the superchain registry network name. Empty for chains that are not in
	// the registry.
	SuperchainName string
	// Versions maps contract release tags to their deployed contract data. Only the
	// OPContractsManager entry is required by OP Deployer.
	Versions validation.Versions
	// Roles are the standard role addresses for the chain.
	Roles validation.RolesConfig
	// SuperchainProxyAdmin is the address of the Superchain ProxyAdmin.
	SuperchainProxyAdmin common.Address
	// ExplorerAPIURL is an Etherscan-compatible API endpoint used by the verify command. Empty
	// when the chain has no known explorer.
	ExplorerAPIURL string
}

// L1Profiles returns the L1 profiles available to this process, keyed by chain ID. Ethereum
// mainnet and Sepolia are always present and are sourced from the superchain registry. A custom
// chain is added when CUSTOM_CHAIN_ID is set.
//
// The registry is rebuilt on every call so that the process always reflects the current
// environment.
func L1Profiles() (map[uint64]L1Profile, error) {
	profiles := map[uint64]L1Profile{
		1: {
			Name:           "mainnet",
			ChainID:        1,
			SuperchainName: "mainnet",
			Versions:       validation.StandardVersionsMainnet,
			Roles:          validation.StandardConfigRolesMainnet,
			// DO NOT MODIFY THIS ADDRESS WITHOUT CLEARING IT WITH THE EVM SAFETY TEAM.
			SuperchainProxyAdmin: common.HexToAddress("0x543bA4AADBAb8f9025686Bd03993043599c6fB04"),
			ExplorerAPIURL:       "https://api.etherscan.io/api",
		},
		11155111: {
			Name:           "sepolia",
			ChainID:        11155111,
			SuperchainName: "sepolia",
			Versions:       validation.StandardVersionsSepolia,
			Roles:          validation.StandardConfigRolesSepolia,
			// DO NOT MODIFY THIS ADDRESS WITHOUT CLEARING IT WITH THE EVM SAFETY TEAM.
			SuperchainProxyAdmin: common.HexToAddress("0x189aBAAaa82DfC015A588A7dbaD6F13b1D3485Bc"),
			ExplorerAPIURL:       "https://api-sepolia.etherscan.io/api",
		},
	}

	custom, err := customL1Profile()
	if err != nil {
		return nil, fmt.Errorf("invalid custom L1 chain configuration: %w", err)
	}
	if custom != nil {
		if _, ok := profiles[custom.ChainID]; ok {
			return nil, fmt.Errorf("invalid custom L1 chain configuration: %s=%d shadows a built-in chain", CustomChainIDEnvVar, custom.ChainID)
		}
		profiles[custom.ChainID] = *custom
	}

	return profiles, nil
}

// L1ProfileFor returns the profile for the given chain ID.
func L1ProfileFor(chainID uint64) (L1Profile, error) {
	profiles, err := L1Profiles()
	if err != nil {
		return L1Profile{}, err
	}
	profile, ok := profiles[chainID]
	if !ok {
		return L1Profile{}, fmt.Errorf("%w: %d", ErrUnsupportedChainID, chainID)
	}
	return profile, nil
}

// customL1Profile builds the custom L1 profile from the environment. It returns nil when
// CUSTOM_CHAIN_ID is unset or zero.
func customL1Profile() (*L1Profile, error) {
	chainID, err := envUint64(CustomChainIDEnvVar)
	if err != nil {
		return nil, err
	}
	if chainID == 0 {
		return nil, nil
	}

	name := strings.TrimSpace(os.Getenv(CustomChainNameEnvVar))
	if name == "" {
		name = fmt.Sprintf("custom-%d", chainID)
	}

	var roles validation.RolesConfig
	roleVars := []struct {
		envVar string
		dst    *validation.Address
	}{
		{CustomChainGuardianEnvVar, &roles.Guardian},
		{CustomChainChallengerEnvVar, &roles.Challenger},
		{CustomChainL1ProxyAdminOwnerEnvVar, &roles.L1ProxyAdminOwner},
		{CustomChainL2ProxyAdminOwnerEnvVar, &roles.L2ProxyAdminOwner},
		{CustomChainProtocolVersionsOwnerVar, &roles.ProtocolVersionsOwner},
	}
	for _, roleVar := range roleVars {
		addr, err := envAddress(roleVar.envVar)
		if err != nil {
			return nil, err
		}
		*roleVar.dst = validation.Address(addr)
	}

	superchainProxyAdmin, err := envAddress(CustomChainSuperchainProxyAdminVar)
	if err != nil {
		return nil, err
	}

	tag := strings.TrimSpace(os.Getenv(CustomChainOPCMTagEnvVar))
	if tag == "" {
		tag = CurrentTag
	}
	opcmImpl, err := envAddress(CustomChainOPCMImplEnvVar)
	if err != nil {
		return nil, err
	}
	// The version config is always registered so that the tag itself is considered supported.
	// A missing OPCM address is reported when something actually asks for it, see
	// OPCMImplAddressFor.
	versionConfig := validation.VersionConfig{}
	if opcmImpl != (common.Address{}) {
		implAddr := validation.Address(opcmImpl)
		versionConfig.OPContractsManager = &validation.ContractData{
			Version:               tag,
			ImplementationAddress: &implAddr,
		}
	}

	return &L1Profile{
		Name:                 name,
		ChainID:              chainID,
		Versions:             validation.Versions{validation.Semver(tag): versionConfig},
		Roles:                roles,
		SuperchainProxyAdmin: superchainProxyAdmin,
		ExplorerAPIURL:       strings.TrimSpace(os.Getenv(CustomChainExplorerAPIURLEnvVar)),
	}, nil
}

// envUint64 reads an optional uint64 from the environment. An unset or blank variable returns 0.
func envUint64(envVar string) (uint64, error) {
	raw := strings.TrimSpace(os.Getenv(envVar))
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s is not a valid chain ID: %q", envVar, raw)
	}
	return value, nil
}

// envAddress reads an optional address from the environment. An unset or blank variable returns
// the zero address.
func envAddress(envVar string) (common.Address, error) {
	raw := strings.TrimSpace(os.Getenv(envVar))
	if raw == "" {
		return common.Address{}, nil
	}
	if !common.IsHexAddress(raw) {
		return common.Address{}, fmt.Errorf("%s is not a valid address: %q", envVar, raw)
	}
	return common.HexToAddress(raw), nil
}
