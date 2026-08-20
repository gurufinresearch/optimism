package script

import "github.com/ethereum/go-ethereum/common"

// DefaultDeterministicDeployerAddress is the canonical address of the Arachnid deterministic
// deployment proxy. It is the address the proxy has on Ethereum, and on every chain that has
// replayed the standard presigned ("Nick's method") deployment transaction.
var DefaultDeterministicDeployerAddress = common.HexToAddress("0x4e59b44847b379578588920ca78fbf26c0b4956c")

var (
	// DeterministicDeployerAddress is the address of the deterministic deployer Forge uses
	// to provide deterministic contract addresses.
	//
	// It defaults to the canonical address and may be overridden with
	// SetDeterministicDeployerAddress for L1 chains that carry the proxy somewhere else. Every
	// consumer reads this variable, so an override keeps the address used to predict CREATE2
	// results (the prank sender and vm.computeCreate2Address) in step with the address that
	// broadcast CREATE2 calls are actually sent to.
	DeterministicDeployerAddress = DefaultDeterministicDeployerAddress
)

// SetDeterministicDeployerAddress overrides the deterministic deployer address for this process.
// Passing the zero address restores the canonical default.
//
// This must be called before any Host is constructed or any broadcast is assembled, otherwise
// predicted and actual CREATE2 addresses can disagree. op-deployer calls it once from its CLI
// Before hook.
func SetDeterministicDeployerAddress(addr common.Address) {
	if addr == (common.Address{}) {
		DeterministicDeployerAddress = DefaultDeterministicDeployerAddress
		return
	}
	DeterministicDeployerAddress = addr
}
