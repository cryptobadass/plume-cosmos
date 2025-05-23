package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/x/bank/types"
)

// RegisterInvariants registers the bank module invariants
func RegisterInvariants(ir sdk.InvariantRegistry, k Keeper) {
	ir.RegisterRoute(types.ModuleName, "nonnegative-outstanding", NonnegativeBalanceInvariant(k))
	ir.RegisterRoute(types.ModuleName, "total-supply", TotalSupply(k))
}

// AllInvariants runs all invariants of the X/bank module.
func AllInvariants(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		return TotalSupply(k)(ctx)
	}
}

// NonnegativeBalanceInvariant checks that all accounts in the application have non-negative balances
func NonnegativeBalanceInvariant(k ViewKeeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var (
			msg   string
			count int
		)

		k.IterateAllBalances(ctx, func(addr sdk.AccAddress, balance sdk.Coin) bool {
			if balance.IsNegative() {
				count++
				msg += fmt.Sprintf("\t%s has a negative balance of %s\n", addr, balance)
			}

			return false
		})
		k.IterateAllWeiBalances(ctx, func(addr sdk.AccAddress, balance sdk.Int) bool {
			if balance.IsNegative() {
				count++
				msg += fmt.Sprintf("\t%s has a negative wei balance of %s\n", addr, balance)
			}

			return false
		})

		broken := count != 0

		return sdk.FormatInvariant(
			types.ModuleName, "nonnegative-outstanding",
			fmt.Sprintf("amount of negative balances found %d\n%s", count, msg),
		), broken
	}
}

// TotalSupply checks that the total supply reflects all the coins held in accounts
func TotalSupply(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		expectedTotal := sdk.Coins{}
		weiTotal := sdk.NewInt(0)
		supply, _, err := k.GetPaginatedTotalSupply(ctx, &query.PageRequest{Limit: query.MaxLimit})

		if err != nil {
			return sdk.FormatInvariant(types.ModuleName, "query supply",
				fmt.Sprintf("error querying total supply %v", err)), false
		}

		k.IterateAllBalances(ctx, func(_ sdk.AccAddress, balance sdk.Coin) bool {
			expectedTotal = expectedTotal.Add(balance)
			return false
		})
		// also iterate over deferred balances
		k.IterateDeferredBalances(ctx, func(addr sdk.AccAddress, coin sdk.Coin) bool {
			expectedTotal = expectedTotal.Add(coin)
			return false
		})
		k.IterateAllWeiBalances(ctx, func(addr sdk.AccAddress, balance sdk.Int) bool {
			weiTotal = weiTotal.Add(balance)
			return false
		})
		weiInUplume, weiRemainder := SplitUplumeWeiAmount(weiTotal)
		if !weiRemainder.IsZero() {
			return sdk.FormatInvariant(types.ModuleName, "total supply",
				fmt.Sprintf(
					"\twei remainder: %v\n",
					weiRemainder)), true
		}
		baseDenom, err := sdk.GetBaseDenom()
		if err == nil {
			expectedTotal = expectedTotal.Add(sdk.NewCoin(baseDenom, weiInUplume))
		} else if !weiInUplume.IsZero() {
			return sdk.FormatInvariant(types.ModuleName, "total supply", "non-zero wei balance without base denom"), true
		}

		broken := !expectedTotal.IsEqual(supply)

		return sdk.FormatInvariant(types.ModuleName, "total supply",
			fmt.Sprintf(
				"\tsum of accounts coins: %v\n"+
					"\tsupply.Total:          %v\n",
				expectedTotal, supply)), broken
	}
}
