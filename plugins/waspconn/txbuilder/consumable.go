// package to build value transaction
package txbuilder

import (
	"github.com/iotaledger/goshimmer/packages/ledgerstate"
	"golang.org/x/xerrors"
)

type ConsumableOutput struct {
	output   ledgerstate.Output
	remain   map[ledgerstate.Color]uint64
	consumed map[ledgerstate.Color]uint64
}

func NewConsumableOutput(out ledgerstate.Output) *ConsumableOutput {
	ret := &ConsumableOutput{
		output:   out,
		remain:   make(map[ledgerstate.Color]uint64),
		consumed: make(map[ledgerstate.Color]uint64),
	}
	out.Balances().ForEach(func(col ledgerstate.Color, bal uint64) bool {
		ret.remain[col] = bal
		return true
	})
	return ret
}

func (o *ConsumableOutput) Clone() *ConsumableOutput {
	ret := &ConsumableOutput{
		output:   o.output.Clone(),
		remain:   make(map[ledgerstate.Color]uint64),
		consumed: make(map[ledgerstate.Color]uint64),
	}
	for col, bal := range o.remain {
		ret.remain[col] = bal
	}
	for col, bal := range o.consumed {
		ret.consumed[col] = bal
	}
	return ret
}

func (o *ConsumableOutput) ConsumableBalance(color ledgerstate.Color) uint64 {
	ret, _ := o.remain[color]
	return ret
}

func (o *ConsumableOutput) WasConsumed() bool {
	return len(o.consumed) > 0
}

func ConsumableBalance(color ledgerstate.Color, consumables ...*ConsumableOutput) uint64 {
	ret := uint64(0)
	for _, out := range consumables {
		ret += out.ConsumableBalance(color)
	}
	return ret
}

// ConsumeColoredTokens specified amount of colored tokens sequentially from specified ConsumableOutputs
// return nil if it was a success.
// In case of failure ConsumableOutputs remain unchanged
func ConsumeColoredTokens(addTo map[ledgerstate.Color]uint64, color ledgerstate.Color, amount uint64, consumables ...*ConsumableOutput) error {
	consumable := ConsumableBalance(color, consumables...)
	if consumable < amount {
		return xerrors.New("ConsumeColoredTokens: not enough balance")
	}
	remaining := amount
	for _, out := range consumables {
		if remaining == 0 {
			break
		}
		rem, _ := out.remain[color]
		if rem == 0 {
			continue
		}
		cons, _ := out.consumed[color]
		if rem >= remaining {
			out.remain[color] = rem - remaining
			remaining = 0
			out.consumed[color] = cons + remaining
		} else {
			out.remain[color] = 0
			remaining -= rem
			out.consumed[color] = cons + rem
		}
	}
	if remaining != 0 {
		panic("ConsumeColoredTokens: internal error")
	}
	s, _ := addTo[color]
	addTo[color] = s + amount
	return nil
}

func ConsumeIOTA(amount uint64, consumables ...*ConsumableOutput) error {
	addTo := make(map[ledgerstate.Color]uint64)
	return ConsumeColoredTokens(addTo, ledgerstate.ColorIOTA, amount, consumables...)
}

// ConsumeRemaining consumes all remaining tokens and return map of consumed balances
func ConsumeRemaining(addTo map[ledgerstate.Color]uint64, consumables ...*ConsumableOutput) {
	for _, out := range consumables {
		for col, bal := range out.remain {
			cons, _ := out.consumed[col]
			out.consumed[col] = cons + bal
			total, _ := addTo[col]
			addTo[col] = total + bal
		}
		out.remain = make(map[ledgerstate.Color]uint64) // clear remaining
	}
}

func SelectConsumed(consumables ...*ConsumableOutput) []*ConsumableOutput {
	ret := make([]*ConsumableOutput, 0)
	for _, out := range consumables {
		if out.WasConsumed() {
			ret = append(ret, out)
		}
	}
	return ret
}

func MakeUTXOInputs(consumables ...*ConsumableOutput) ledgerstate.Inputs {
	ret := make(ledgerstate.Inputs, len(consumables))
	for i, out := range consumables {
		ret[i] = ledgerstate.NewUTXOInput(out.output.ID())
	}
	return ret
}
