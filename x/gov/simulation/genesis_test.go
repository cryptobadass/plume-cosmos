package simulation_test

import (
	"encoding/json"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/gov/simulation"
	"github.com/cosmos/cosmos-sdk/x/gov/types"
)

// TestRandomizedGenState tests the normal scenario of applying RandomizedGenState.
// Abonormal scenarios are not tested here.
func TestRandomizedGenState(t *testing.T) {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(interfaceRegistry)

	s := rand.NewSource(1)
	r := rand.New(s)

	simState := module.SimulationState{
		AppParams:    make(simtypes.AppParams),
		Cdc:          cdc,
		Rand:         r,
		NumBonded:    3,
		Accounts:     simtypes.RandomAccounts(r, 3),
		InitialStake: 1000,
		GenState:     make(map[string]json.RawMessage),
	}

	simulation.RandomizedGenState(&simState)

	var govGenesis types.GenesisState
	simState.Cdc.MustUnmarshalJSON(simState.GenState[types.ModuleName], &govGenesis)

	dec1, _ := sdk.NewDecFromStr("0.375000000000000000")
	dec2, _ := sdk.NewDecFromStr("0.474000000000000000")
	dec3, _ := sdk.NewDecFromStr("0.291000000000000000")
	dec4, _ := sdk.NewDecFromStr("0.528000000000000000")
	dec5, _ := sdk.NewDecFromStr("0.511000000000000000")

	require.Equal(t, "272uplume", govGenesis.DepositParams.MinDeposit.String())
	require.Equal(t, "42h4m57s", govGenesis.DepositParams.MaxDepositPeriod.String())
	require.Equal(t, "800uplume", govGenesis.DepositParams.MinExpeditedDeposit.String())
	require.Equal(t, float64(97711), govGenesis.VotingParams.VotingPeriod.Seconds())
	require.Equal(t, float64(53488), govGenesis.VotingParams.ExpeditedVotingPeriod.Seconds())
	require.Equal(t, dec1, govGenesis.TallyParams.Quorum)
	require.Equal(t, dec2, govGenesis.TallyParams.Threshold)
	require.Equal(t, dec3, govGenesis.TallyParams.VetoThreshold)
	require.Equal(t, dec4, govGenesis.TallyParams.ExpeditedQuorum)
	require.Equal(t, dec5, govGenesis.TallyParams.ExpeditedThreshold)
	require.Equal(t, uint64(0x28), govGenesis.StartingProposalId)
	require.Equal(t, types.Deposits{}, govGenesis.Deposits)
	require.Equal(t, types.Votes{}, govGenesis.Votes)
	require.Equal(t, types.Proposals{}, govGenesis.Proposals)
}

// TestRandomizedGenState tests abnormal scenarios of applying RandomizedGenState.
func TestRandomizedGenState1(t *testing.T) {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(interfaceRegistry)

	s := rand.NewSource(1)
	r := rand.New(s)
	// all these tests will panic
	tests := []struct {
		simState module.SimulationState
		panicMsg string
	}{
		{ // panic => reason: incomplete initialization of the simState
			module.SimulationState{}, "invalid memory address or nil pointer dereference"},
		{ // panic => reason: incomplete initialization of the simState
			module.SimulationState{
				AppParams: make(simtypes.AppParams),
				Cdc:       cdc,
				Rand:      r,
			}, "assignment to entry in nil map"},
	}

	for _, tt := range tests {
		require.Panicsf(t, func() { simulation.RandomizedGenState(&tt.simState) }, tt.panicMsg)
	}
}
