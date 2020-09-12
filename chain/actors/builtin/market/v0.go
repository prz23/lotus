package market

import (
	"github.com/filecoin-project/specs-actors/actors/builtin/market"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
)

type v0State struct {
	market.State
	store adt.Store
}

func (s *v0State) TotalLocked() (abi.TokenAmount, error) {
	fml := types.BigAdd(s.TotalClientLockedCollateral, s.TotalProviderLockedCollateral)
	fml = types.BigAdd(fml, s.TotalClientStorageFee)
	return fml, nil
}

func (s *v0State) EscrowTable() (BalanceTable, error) {
	return adt.AsBalanceTable(s.store, s.State.EscrowTable)
}

func (s *v0State) Lockedtable() (BalanceTable, error) {
	return adt.AsBalanceTable(s.store, s.State.LockedTable)
}