package main

import (
	"os"

	"github.com/cosmos/cosmos-sdk/server"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/cosmos/cosmos-sdk/simapp"
	"github.com/cosmos/cosmos-sdk/simapp/simd/cmd"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func main() {
	// Set custom bech32 address prefixes for Plume
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("plume", "plumepub")
	config.SetBech32PrefixForValidator("plumevaloper", "plumevaloperpub")
	config.SetBech32PrefixForConsensusNode("plumevalcons", "plumevalconspub")
	config.Seal()

	rootCmd, _ := cmd.NewRootCmd()

	if err := svrcmd.Execute(rootCmd, simapp.DefaultNodeHome); err != nil {
		switch e := err.(type) {
		case server.ErrorCode:
			os.Exit(e.Code)

		default:
			os.Exit(1)
		}
	}
}
