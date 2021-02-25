package utxodb

import (
	"github.com/iotaledger/goshimmer/packages/ledgerstate"
	"golang.org/x/xerrors"
)

func (u *UtxoDB) collectInputBalances(tx *ledgerstate.Transaction) (map[ledgerstate.Color]uint64, uint64, error) {
	ret := make(map[ledgerstate.Color]uint64)
	retsum := uint64(0)

	for _, inp := range tx.Essence().Inputs() {
		if inp.Type() != ledgerstate.UTXOInputType {
			return nil, 0, xerrors.New("wrong input type")
		}
		utxoInp := inp.(*ledgerstate.UTXOInput)
		out, ok := u.utxo[utxoInp.ReferencedOutputID()]
		if !ok {
			panic("collectInputBalances: output does not exist")
		}
		out.Balances().ForEach(func(col ledgerstate.Color, bal uint64) bool {
			s, _ := ret[col]
			ret[col] = s + bal
			retsum += bal
			return true
		})
	}
	return ret, retsum, nil
}

func collectOutputBalances(tx *ledgerstate.Transaction) (map[ledgerstate.Color]uint64, uint64) {
	ret := make(map[ledgerstate.Color]uint64)
	retsum := uint64(0)

	for _, out := range tx.Essence().Outputs() {
		out.Balances().ForEach(func(col ledgerstate.Color, bal uint64) bool {
			s, _ := ret[col]
			ret[col] = s + bal
			retsum += bal
			return true
		})
	}
	return ret, retsum
}

func (u *UtxoDB) CheckInputsOutputs(tx *ledgerstate.Transaction) error {
	inbals, insum, err := u.collectInputBalances(tx)
	if err != nil {
		return xerrors.Errorf("utxodb.CheckInputsOutputs: wrong inputs: %v", err)
	}
	outbals, outsum := collectOutputBalances(tx)
	if insum != outsum {
		return xerrors.New("utxodb.CheckInputsOutputs unequal totals")
	}

	for col, inb := range inbals {
		if col == ledgerstate.ColorMint {
			return xerrors.New("utxodb.CheckInputsOutputs: assertion failed: input cannot ")
		}
		if col == ledgerstate.ColorIOTA {
			continue
		}
		outb, ok := outbals[col]
		if !ok {
			continue
		}
		if outb > inb {
			// colored supply can't be inflated
			return xerrors.New("utxodb.CheckInputsOutputs: colored supply can't be inflated")
		}
	}
	return nil
}
