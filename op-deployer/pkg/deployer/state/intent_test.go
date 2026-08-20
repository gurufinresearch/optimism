package state

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestValidateStandardValues(t *testing.T) {
	intent, err := NewIntentStandard(11155111, []common.Hash{common.HexToHash("0x336")})
	require.NoError(t, err)

	err = intent.Check()
	require.Error(t, err)
	require.ErrorIs(t, err, addresses.ErrZeroAddress)

	setChainRoles(&intent)
	err = intent.Check()
	require.Error(t, err)
	require.ErrorIs(t, err, ErrFeeVaultZeroAddress)

	setFeeAddresses(&intent)
	err = intent.Check()
	require.NoError(t, err)

	tests := []struct {
		name    string
		mutator func(intent *Intent)
		err     error
	}{
		{
			"EIP1559Denominator",
			func(intent *Intent) {
				intent.Chains[0].Eip1559Denominator = 3
			},
			ErrNonStandardValue,
		},
		{
			"EIP1559DenominatorCanyon",
			func(intent *Intent) {
				intent.Chains[0].Eip1559DenominatorCanyon = 3
			},
			ErrNonStandardValue,
		},
		{
			"EIP1559Elasticity",
			func(intent *Intent) {
				intent.Chains[0].Eip1559Elasticity = 999
			},
			ErrNonStandardValue,
		},
		{
			"AdditionalDisputeGames",
			func(intent *Intent) {
				intent.Chains[0].AdditionalDisputeGames = []AdditionalDisputeGame{
					{
						VMType: VMTypeAlphabet,
					},
				}
			},
			ErrNonStandardValue,
		},
		{
			"SuperchainConfigProxy",
			func(intent *Intent) {
				addr := common.HexToAddress("0x9999")
				intent.SuperchainConfigProxy = &addr
			},
			ErrIncompatibleValue,
		},
		{
			"OPCMAddress",
			func(intent *Intent) {
				addr := common.HexToAddress("0x9999")
				intent.OPCMAddress = &addr
			},
			ErrNonStandardValue,
		},
		{
			"SuperchainRoles",
			func(intent *Intent) {
				intent.SuperchainRoles = &addresses.SuperchainRoles{
					SuperchainGuardian: common.HexToAddress("0x9999"),
				}
			},
			ErrIncompatibleValue,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent, err := NewIntentStandard(11155111, []common.Hash{common.HexToHash("0x336")})
			require.NoError(t, err)
			setChainRoles(&intent)
			setFeeAddresses(&intent)

			tt.mutator(&intent)

			err = intent.Check()
			require.Error(t, err)
			require.ErrorIs(t, err, tt.err)
		})
	}
}

func TestValidateCustomValues(t *testing.T) {
	intent, err := NewIntentCustom(1, []common.Hash{common.HexToHash("0x336")})
	require.NoError(t, err)

	err = intent.Check()
	require.Error(t, err)
	require.ErrorIs(t, err, addresses.ErrZeroAddress)

	setSuperchainRoles(&intent)
	err = intent.Check()
	require.Error(t, err)
	require.ErrorIs(t, err, addresses.ErrZeroAddress)

	setChainRoles(&intent)
	err = intent.Check()
	require.Error(t, err)
	require.ErrorIs(t, err, ErrEip1559ZeroValue)

	setEip1559Params(&intent)
	err = intent.Check()
	require.Error(t, err)
	require.ErrorIs(t, err, ErrFeeVaultZeroAddress)

	setFeeAddresses(&intent)
	err = intent.Check()
	require.NoError(t, err)

	tests := []struct {
		name    string
		mutator func(intent *Intent)
		err     error
	}{
		{
			"both OPCM and SuperchainRoles defined",
			func(intent *Intent) {
				addr := common.HexToAddress("0x9999")
				intent.SuperchainRoles = &addresses.SuperchainRoles{
					SuperchainGuardian: addr,
				}
				intent.OPCMAddress = &addr
			},
			ErrIncompatibleValue,
		},
		{
			"neither OPCM or SuperchainRoles defined",
			func(intent *Intent) {
				intent.OPCMAddress = nil
				intent.SuperchainRoles = nil
			},
			ErrIncompatibleValue,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent, err := NewIntentCustom(11155111, []common.Hash{common.HexToHash("0x336")})
			require.NoError(t, err)

			setSuperchainRoles(&intent)
			setChainRoles(&intent)

			setEip1559Params(&intent)
			setFeeAddresses(&intent)

			tt.mutator(&intent)

			err = intent.Check()
			require.Error(t, err)
			require.ErrorIs(t, err, tt.err)
		})
	}
}

// TestNewIntentStandardCustomL1 covers `op-deployer init --intent-type standard-overrides`
// against an L1 that is not in the superchain registry, configured via CUSTOM_CHAIN_*.
func TestNewIntentStandardCustomL1(t *testing.T) {
	const (
		customChainID = uint64(4242)
		opcmImpl      = "0x23f49a3c87840405aeeaef5AD6bEE0c503C43345"
		challenger    = "0x2222222222222222222222222222222222222222"
		l1PAO         = "0x3333333333333333333333333333333333333333"
		l2PAO         = "0x4444444444444444444444444444444444444444"
	)

	t.Setenv(standard.CustomChainIDEnvVar, "4242")
	t.Setenv(standard.CustomChainNameEnvVar, "guru-testnet")
	t.Setenv(standard.CustomChainOPCMImplEnvVar, opcmImpl)
	t.Setenv(standard.CustomChainChallengerEnvVar, challenger)
	t.Setenv(standard.CustomChainL1ProxyAdminOwnerEnvVar, l1PAO)
	t.Setenv(standard.CustomChainL2ProxyAdminOwnerEnvVar, l2PAO)

	intent, err := NewIntent(IntentTypeStandardOverrides, customChainID, []common.Hash{common.HexToHash("0x336")})
	require.NoError(t, err)

	require.Equal(t, customChainID, intent.L1ChainID)
	require.Equal(t, common.HexToAddress(opcmImpl), *intent.OPCMAddress)
	require.Len(t, intent.Chains, 1)
	require.Equal(t, common.HexToAddress(challenger), intent.Chains[0].Roles.Challenger)
	require.Equal(t, common.HexToAddress(l1PAO), intent.Chains[0].Roles.L1ProxyAdminOwner)
	require.Equal(t, common.HexToAddress(l2PAO), intent.Chains[0].Roles.L2ProxyAdminOwner)
	require.Equal(t, standard.GasLimit, intent.Chains[0].GasLimit)
}

func setSuperchainRoles(intent *Intent) {
	intent.SuperchainRoles = &addresses.SuperchainRoles{
		SuperchainProxyAdminOwner: common.HexToAddress("0xa"),
		ProtocolVersionsOwner:     common.HexToAddress("0xb"),
		SuperchainGuardian:        common.HexToAddress("0xc"),
		Challenger:                common.HexToAddress("0xd"),
	}
}

func setEip1559Params(intent *Intent) {
	intent.Chains[0].Eip1559Denominator = 5000
	intent.Chains[0].Eip1559DenominatorCanyon = 5000
	intent.Chains[0].Eip1559Elasticity = 5000
}

func setChainRoles(intent *Intent) {
	if intent.Chains[0].Roles.L1ProxyAdminOwner == (common.Address{}) {
		intent.Chains[0].Roles.L1ProxyAdminOwner = common.HexToAddress("0x01")
	}
	if intent.Chains[0].Roles.Challenger == (common.Address{}) {
		intent.Chains[0].Roles.Challenger = common.HexToAddress("0x07")
	}

	intent.Chains[0].Roles.L2ProxyAdminOwner = common.HexToAddress("0x02")
	intent.Chains[0].Roles.SystemConfigOwner = common.HexToAddress("0x03")
	intent.Chains[0].Roles.UnsafeBlockSigner = common.HexToAddress("0x04")
	intent.Chains[0].Roles.Batcher = common.HexToAddress("0x05")
	intent.Chains[0].Roles.Proposer = common.HexToAddress("0x06")
}

func setFeeAddresses(intent *Intent) {
	intent.Chains[0].BaseFeeVaultRecipient = common.HexToAddress("0x08")
	intent.Chains[0].L1FeeVaultRecipient = common.HexToAddress("0x09")
	intent.Chains[0].SequencerFeeVaultRecipient = common.HexToAddress("0x0A")
}
