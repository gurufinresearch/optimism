package standard

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/superchain-registry/validation"

	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/require"
)

const (
	testCustomChainID   = 4242
	testOPCMImplAddr    = "0x23f49a3c87840405aeeaef5AD6bEE0c503C43345"
	testRoleAddr        = "0x498B5AEC5D439B733DC2F58AB489783A23FB26DA"
	testGuardianAddr    = "0x1111111111111111111111111111111111111111"
	testChallengerAddr  = "0x2222222222222222222222222222222222222222"
	testProxyAdminAddr  = "0x3333333333333333333333333333333333333333"
	testExplorerAPIAddr = "https://explorer.guru.example/api"
	testCreate2Deployer = "0x4444444444444444444444444444444444444444"
)

// clearCustomChainEnv makes the custom chain env vars deterministic for a test, regardless of
// what the developer has exported in their shell.
func clearCustomChainEnv(t *testing.T) {
	for _, envVar := range []string{
		CustomChainIDEnvVar,
		CustomChainNameEnvVar,
		CustomChainOPCMImplEnvVar,
		CustomChainOPCMTagEnvVar,
		CustomChainGuardianEnvVar,
		CustomChainChallengerEnvVar,
		CustomChainL1ProxyAdminOwnerEnvVar,
		CustomChainL2ProxyAdminOwnerEnvVar,
		CustomChainProtocolVersionsOwnerVar,
		CustomChainSuperchainProxyAdminVar,
		CustomChainExplorerAPIURLEnvVar,
		CustomChainCreate2DeployerEnvVar,
	} {
		t.Setenv(envVar, "")
	}
}

func TestL1Profiles_BuiltInsOnly(t *testing.T) {
	clearCustomChainEnv(t)

	profiles, err := L1Profiles()
	require.NoError(t, err)
	require.Len(t, profiles, 2)

	mainnet := profiles[1]
	require.Equal(t, "mainnet", mainnet.SuperchainName)
	require.Equal(t, validation.StandardConfigRolesMainnet, mainnet.Roles)
	require.Equal(t, validation.StandardVersionsMainnet, mainnet.Versions)

	require.Equal(t, script.DefaultDeterministicDeployerAddress, mainnet.Create2Deployer)

	sepolia := profiles[11155111]
	require.Equal(t, "sepolia", sepolia.SuperchainName)
	require.Equal(t, validation.StandardConfigRolesSepolia, sepolia.Roles)
	require.Equal(t, validation.StandardVersionsSepolia, sepolia.Versions)
	require.Equal(t, script.DefaultDeterministicDeployerAddress, sepolia.Create2Deployer)

	_, err = L1ProfileFor(testCustomChainID)
	require.ErrorContains(t, err, "unsupported chain ID: 4242")
}

func TestL1Profiles_CustomChain(t *testing.T) {
	clearCustomChainEnv(t)
	t.Setenv(CustomChainIDEnvVar, "4242")
	t.Setenv(CustomChainNameEnvVar, "guru-testnet")
	t.Setenv(CustomChainOPCMImplEnvVar, testOPCMImplAddr)
	t.Setenv(CustomChainGuardianEnvVar, testGuardianAddr)
	t.Setenv(CustomChainChallengerEnvVar, testChallengerAddr)
	t.Setenv(CustomChainL1ProxyAdminOwnerEnvVar, testRoleAddr)
	t.Setenv(CustomChainL2ProxyAdminOwnerEnvVar, testRoleAddr)
	t.Setenv(CustomChainProtocolVersionsOwnerVar, testRoleAddr)
	t.Setenv(CustomChainSuperchainProxyAdminVar, testProxyAdminAddr)
	t.Setenv(CustomChainExplorerAPIURLEnvVar, testExplorerAPIAddr)
	t.Setenv(CustomChainCreate2DeployerEnvVar, testCreate2Deployer)

	profiles, err := L1Profiles()
	require.NoError(t, err)
	require.Len(t, profiles, 3)

	profile, err := L1ProfileFor(testCustomChainID)
	require.NoError(t, err)
	require.Equal(t, "guru-testnet", profile.Name)
	require.Equal(t, uint64(testCustomChainID), profile.ChainID)
	require.Empty(t, profile.SuperchainName)
	require.Equal(t, testExplorerAPIAddr, profile.ExplorerAPIURL)

	// The accessors used by the standard intent types must all resolve.
	opcm, err := OPCMImplAddressFor(testCustomChainID, CurrentTag)
	require.NoError(t, err)
	require.Equal(t, common.HexToAddress(testOPCMImplAddr), opcm)

	guardian, err := GuardianAddressFor(testCustomChainID)
	require.NoError(t, err)
	require.Equal(t, common.HexToAddress(testGuardianAddr), guardian)

	challenger, err := ChallengerAddressFor(testCustomChainID)
	require.NoError(t, err)
	require.Equal(t, common.HexToAddress(testChallengerAddr), challenger)

	l1PAO, err := L1ProxyAdminOwner(testCustomChainID)
	require.NoError(t, err)
	require.Equal(t, common.HexToAddress(testRoleAddr), l1PAO)

	l2PAO, err := L2ProxyAdminOwner(testCustomChainID)
	require.NoError(t, err)
	require.Equal(t, common.HexToAddress(testRoleAddr), l2PAO)

	pvo, err := ProtocolVersionsOwner(testCustomChainID)
	require.NoError(t, err)
	require.Equal(t, common.HexToAddress(testRoleAddr), pvo)

	superchainPA, err := SuperchainProxyAdminAddrFor(testCustomChainID)
	require.NoError(t, err)
	require.Equal(t, common.HexToAddress(testProxyAdminAddr), superchainPA)

	versions, err := L1VersionsFor(testCustomChainID)
	require.NoError(t, err)
	require.Contains(t, versions, validation.Semver(CurrentTag))

	// The custom chain is not in the superchain registry, so this must fail loudly rather than
	// silently returning mainnet's superchain config.
	_, err = SuperchainFor(testCustomChainID)
	require.ErrorContains(t, err, "not in the superchain registry")

	// Built-ins are unaffected.
	sepoliaChallenger, err := ChallengerAddressFor(11155111)
	require.NoError(t, err)
	require.Equal(t, common.Address(validation.StandardConfigRolesSepolia.Challenger), sepoliaChallenger)
}

func TestL1Profiles_CustomChainDefaults(t *testing.T) {
	clearCustomChainEnv(t)
	t.Setenv(CustomChainIDEnvVar, "4242")

	profile, err := L1ProfileFor(testCustomChainID)
	require.NoError(t, err)
	require.Equal(t, "custom-4242", profile.Name)
	require.Equal(t, common.Address{}, common.Address(profile.Roles.Guardian))
	require.Equal(t, common.Address{}, profile.SuperchainProxyAdmin)

	// The tag is registered even without an OPCM address, so checkL1Prod gets past its tag check
	// and the error names the variable that needs to be set.
	versions, err := L1VersionsFor(testCustomChainID)
	require.NoError(t, err)
	require.Contains(t, versions, validation.Semver(CurrentTag))

	_, err = OPCMImplAddressFor(testCustomChainID, CurrentTag)
	require.ErrorContains(t, err, CustomChainOPCMImplEnvVar)

	// Roles that were never set come back as the zero address rather than an error. The intent
	// checks reject those at apply time.
	guardian, err := GuardianAddressFor(testCustomChainID)
	require.NoError(t, err)
	require.Equal(t, common.Address{}, guardian)
}

func TestL1Profiles_CustomChainOPCMTag(t *testing.T) {
	clearCustomChainEnv(t)
	t.Setenv(CustomChainIDEnvVar, "4242")
	t.Setenv(CustomChainOPCMTagEnvVar, ContractsV400Tag)
	t.Setenv(CustomChainOPCMImplEnvVar, testOPCMImplAddr)

	opcm, err := OPCMImplAddressFor(testCustomChainID, ContractsV400Tag)
	require.NoError(t, err)
	require.Equal(t, common.HexToAddress(testOPCMImplAddr), opcm)

	_, err = OPCMImplAddressFor(testCustomChainID, CurrentTag)
	require.ErrorContains(t, err, "unsupported tag")
}

func TestL1Profiles_Errors(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		errPart string
	}{
		{
			name:    "shadows mainnet",
			env:     map[string]string{CustomChainIDEnvVar: "1"},
			errPart: "shadows a built-in chain",
		},
		{
			name:    "shadows sepolia",
			env:     map[string]string{CustomChainIDEnvVar: "11155111"},
			errPart: "shadows a built-in chain",
		},
		{
			name:    "malformed chain ID",
			env:     map[string]string{CustomChainIDEnvVar: "0x1234"},
			errPart: CustomChainIDEnvVar + " is not a valid chain ID",
		},
		{
			name: "malformed address",
			env: map[string]string{
				CustomChainIDEnvVar:         "4242",
				CustomChainChallengerEnvVar: "0xnope",
			},
			errPart: CustomChainChallengerEnvVar + " is not a valid address",
		},
		{
			name: "truncated address",
			env: map[string]string{
				CustomChainIDEnvVar:       "4242",
				CustomChainOPCMImplEnvVar: "0x23f49a3c",
			},
			errPart: CustomChainOPCMImplEnvVar + " is not a valid address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearCustomChainEnv(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			_, err := L1Profiles()
			require.ErrorContains(t, err, tt.errPart)

			// Every accessor surfaces the configuration error, including for built-in chains,
			// so a typo can never be mistaken for a working setup. OPCMImplAddressFor is the
			// first call `op-deployer init` makes, and it must not mask the cause behind its
			// generic "unsupported chainID" message.
			_, err = ChallengerAddressFor(1)
			require.ErrorContains(t, err, tt.errPart)
			require.NotErrorIs(t, err, ErrUnsupportedChainID)

			_, err = OPCMImplAddressFor(1, CurrentTag)
			require.ErrorContains(t, err, tt.errPart)
		})
	}
}

func TestCreate2Deployer_FallsBackWhenUnset(t *testing.T) {
	clearCustomChainEnv(t)

	// the resolver alone
	addr, err := Create2DeployerFromEnv()
	require.NoError(t, err)
	require.Equal(t, script.DefaultDeterministicDeployerAddress, addr)

	// and through a custom profile, which must not lose the canonical default
	t.Setenv(CustomChainIDEnvVar, "4242")
	profile, err := L1ProfileFor(testCustomChainID)
	require.NoError(t, err)
	require.Equal(t, script.DefaultDeterministicDeployerAddress, profile.Create2Deployer)
}

func TestCreate2Deployer_Override(t *testing.T) {
	clearCustomChainEnv(t)
	t.Setenv(CustomChainIDEnvVar, "4242")
	t.Setenv(CustomChainCreate2DeployerEnvVar, testCreate2Deployer)

	profile, err := L1ProfileFor(testCustomChainID)
	require.NoError(t, err)
	require.Equal(t, common.HexToAddress(testCreate2Deployer), profile.Create2Deployer)
}

func TestCreate2Deployer_MalformedIsAnError(t *testing.T) {
	clearCustomChainEnv(t)
	t.Setenv(CustomChainCreate2DeployerEnvVar, "not-an-address")

	_, err := Create2DeployerFromEnv()
	require.ErrorContains(t, err, CustomChainCreate2DeployerEnvVar)
}

func TestApplyCreate2DeployerOverride(t *testing.T) {
	original := script.DeterministicDeployerAddress
	t.Cleanup(func() { script.DeterministicDeployerAddress = original })

	// unset -> canonical
	clearCustomChainEnv(t)
	require.NoError(t, ApplyCreate2DeployerOverride())
	require.Equal(t, script.DefaultDeterministicDeployerAddress, script.DeterministicDeployerAddress)

	// set -> override, which every consumer of the package var picks up
	t.Setenv(CustomChainCreate2DeployerEnvVar, testCreate2Deployer)
	require.NoError(t, ApplyCreate2DeployerOverride())
	require.Equal(t, common.HexToAddress(testCreate2Deployer), script.DeterministicDeployerAddress)

	// malformed -> error, and the address is left alone
	t.Setenv(CustomChainCreate2DeployerEnvVar, "0xnope")
	require.Error(t, ApplyCreate2DeployerOverride())
	require.Equal(t, common.HexToAddress(testCreate2Deployer), script.DeterministicDeployerAddress)
}
