// package to build value transaction
package txbuilder

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/iotaledger/goshimmer/packages/ledgerstate"
	"golang.org/x/xerrors"
	"sort"
)

type inputBalances struct {
	output   ledgerstate.Output
	remain   map[ledgerstate.Color]uint64
	consumed map[ledgerstate.Color]uint64
}

type Builder struct {
	addr          ledgerstate.Address
	inputBalances []*inputBalances
	outputs       ledgerstate.Outputs
}

var (
	errorWrongInputs      = xerrors.New("wrong inputs")
	errorWrongColor       = xerrors.New("wrong color")
	errorNotEnoughBalance = xerrors.New("non existent or not enough colored balance")
)

func NewFromAddressOutputs(addr ledgerstate.Address, utxos ledgerstate.Outputs) (*Builder, error) {
	ret := &Builder{
		addr:          addr,
		inputBalances: make([]*inputBalances, len(utxos)),
		outputs:       make(ledgerstate.Outputs, 0),
	}
	for i, out := range utxos {
		if !EqualAddresses(addr, out.Address()) {
			return nil, xerrors.New("expected all outputs with the same address")
		}
		ret.inputBalances[i] = &inputBalances{}
		ret.inputBalances[i].output = out.Clone()
		ret.inputBalances[i].consumed = make(map[ledgerstate.Color]uint64)
		ret.inputBalances[i].remain = make(map[ledgerstate.Color]uint64)
		out.Balances().ForEach(func(col ledgerstate.Color, bal uint64) bool {
			ret.inputBalances[i].remain[col] = bal
			return true
		})
	}
	return ret, nil
}

func (txb *Builder) Clone() *Builder {
	ret := &Builder{
		addr:          txb.addr,
		inputBalances: make([]*inputBalances, len(txb.inputBalances)),
		outputs:       make(ledgerstate.Outputs, len(txb.outputs)),
	}
	for i := range txb.inputBalances {
		ret.inputBalances[i] = &inputBalances{}
		ret.inputBalances[i].output = txb.inputBalances[i].output.Clone()
		ret.inputBalances[i].consumed = make(map[ledgerstate.Color]uint64)
		for col, bal := range txb.inputBalances[i].consumed {
			ret.inputBalances[i].consumed[col] = bal
		}
		ret.inputBalances[i].remain = make(map[ledgerstate.Color]uint64)
		for col, bal := range txb.inputBalances[i].remain {
			ret.inputBalances[i].remain[col] = bal
		}
	}
	return ret
}

// ConsumeFromInputBalances specified amount of colored tokens sequentially from specified inputBalances
// return nil if it was a success.
// In case of failure inputBalances remain unchanged
func (txb *Builder) ConsumeFromInputBalances(targetAddr ledgerstate.Address, color ledgerstate.Color, amount uint64, inputBalances ...*inputBalances) error {
	if amount == 0 {
		return xerrors.New("OutputIOTAFromInputBalances: amount must be positive")
	}
	if len(inputBalances) == 0 {
		inputBalances = txb.inputBalances
	}
	// check if possible
	total := uint64(0)
	for _, inp := range inputBalances {
		rem, _ := inp.remain[color]
		total += rem
		if total >= amount {
			break
		}
	}
	if total < amount {
		return errorNotEnoughBalance
	}
	for _, inp := range inputBalances {
		if amount == 0 {
			break
		}
		rem, _ := inp.remain[color]
		if rem == 0 {
			continue
		}
		cons, _ := inp.consumed[color]
		if rem >= amount {
			inp.remain[color] = rem - amount
			amount = 0
			inp.consumed[color] = cons + amount
		} else {
			inp.remain[color] = 0
			amount -= rem
			inp.consumed[color] = cons + rem
		}
	}
	if amount != 0 {
		panic("OutputIOTAFromInputBalances: internal error")
	}
	return nil
}

// ForEachInputBalance iterates through reminders
func (txb *Builder) ForEachInputBalance(consumer func(oid *valuetransaction.OutputID, bals []*balance.Balance) bool) {
	for i := range txb.inputBalances {
		if !consumer(&txb.inputBalances[i].outputId, txb.inputBalances[i].remain) {
			return
		}
	}
}

func (txb *Builder) sortInputBalancesById() {
	sort.Slice(txb.inputBalances, func(i, j int) bool {
		return bytes.Compare(txb.inputBalances[i].outputId[:], txb.inputBalances[j].outputId[:]) < 0
	})
}

func (txb *Builder) SetConsumerPrioritySmallerBalances() {
	sort.Slice(txb.inputBalances, func(i, j int) bool {
		si := txutil.BalancesSumTotal(txb.inputBalances[i].remain)
		sj := txutil.BalancesSumTotal(txb.inputBalances[j].remain)
		if si == sj {
			return i < j
		}
		return si < sj
	})
}

func (txb *Builder) SetConsumerPriorityLargerBalances() {
	if txb.finalized {
		panic("using finalized transaction builder")
	}
	sort.Slice(txb.inputBalances, func(i, j int) bool {
		si := txutil.BalancesSumTotal(txb.inputBalances[i].remain)
		sj := txutil.BalancesSumTotal(txb.inputBalances[j].remain)
		if si == sj {
			return i < j
		}
		return si > sj
	})
}

// GetInputBalance what is available in inputs
func (txb *Builder) GetInputBalance(col balance.Color) int64 {
	if txb.finalized {
		panic("using finalized transaction builder")
	}
	ret := int64(0)
	for _, inp := range txb.inputBalances {
		ret += txutil.BalanceOfColor(inp.remain, col)
	}
	return ret
}

// Returns consumed and unconsumed total
func subtractAmount(bals []*balance.Balance, col balance.Color, amount int64) (int64, int64) {
	if amount == 0 {
		return 0, 0
	}
	for _, bal := range bals {
		if bal.Color == col {
			if bal.Value >= amount {
				bal.Value -= amount
				return amount, 0
			}
			bal.Value = 0
			return bal.Value, amount - bal.Value
		}
	}
	return 0, amount
}

func addAmount(bals []*balance.Balance, col balance.Color, amount int64) []*balance.Balance {
	if amount == 0 {
		return bals
	}
	for _, bal := range bals {
		if bal.Color == col {
			bal.Value += amount
			return bals
		}
	}
	return append(bals, balance.New(col, amount))
}

// don't do any validation, may panic
func (txb *Builder) moveAmount(targetAddr address.Address, origColor, targetColor balance.Color, amountToConsume int64) {
	saveAmount := amountToConsume
	if amountToConsume == 0 {
		return
	}
	var consumedAmount int64
	for i := range txb.inputBalances {
		consumedAmount, amountToConsume = subtractAmount(txb.inputBalances[i].remain, origColor, amountToConsume)
		txb.inputBalances[i].consumed = addAmount(txb.inputBalances[i].consumed, origColor, consumedAmount)
		if amountToConsume == 0 {
			break
		}
	}
	if amountToConsume > 0 {
		panic(errorNotEnoughBalance)
	}
	txb.addToOutputs(targetAddr, targetColor, saveAmount)
}

func (txb *Builder) moveAmountFromTransaction(targetAddr address.Address, origColor, targetColor balance.Color, amountToConsume int64, txid valuetransaction.ID) {
	saveAmount := amountToConsume
	if amountToConsume == 0 {
		return
	}
	for i := range txb.inputBalances {
		if txb.inputBalances[i].outputId.TransactionID() != txid {
			continue
		}
		var consumedAmount int64
		consumedAmount, amountToConsume = subtractAmount(txb.inputBalances[i].remain, origColor, amountToConsume)
		txb.inputBalances[i].consumed = addAmount(txb.inputBalances[i].consumed, origColor, consumedAmount)
		if amountToConsume == 0 {
			break
		}
	}
	if amountToConsume > 0 {
		panic(errorNotEnoughBalance)
	}
	txb.addToOutputs(targetAddr, targetColor, saveAmount)
}

func (txb *Builder) addToOutputs(targetAddr address.Address, col balance.Color, amount int64) {
	cmap, ok := txb.outputBalances[targetAddr]
	if !ok {
		cmap = make(map[balance.Color]int64)
		txb.outputBalances[targetAddr] = cmap
	}
	b, _ := cmap[col]
	cmap[col] = b + amount
}

// MoveTokensToAddress move token without changing color
func (txb *Builder) MoveTokensToAddress(targetAddr address.Address, col balance.Color, amount int64) error {
	if txb.finalized {
		panic("using finalized transaction builder")
	}
	if txb.GetInputBalance(col) < amount {
		return errorNotEnoughBalance
	}
	txb.moveAmount(targetAddr, col, col, amount)
	return nil
}

func (txb *Builder) EraseColor(targetAddr address.Address, col balance.Color, amount int64) error {
	if txb.finalized {
		panic("using finalized transaction builder")
	}
	actualBalance := txb.GetInputBalance(col)
	if actualBalance < amount {
		return fmt.Errorf("EraseColor: not enough balance: need %d, found %d, color %s",
			amount, actualBalance, col.String())
	}
	txb.moveAmount(targetAddr, col, balance.ColorIOTA, amount)
	return nil
}

// MintColoredTokens creates output of NewColor tokens out of inputs with specified color
func (txb *Builder) MintColoredTokens(targetAddr address.Address, sourceColor balance.Color, amount int64) error {
	if txb.finalized {
		panic("using finalized transaction builder")
	}
	if txb.GetInputBalance(sourceColor) < amount {
		return errorNotEnoughBalance
	}
	txb.moveAmount(targetAddr, sourceColor, balance.ColorNew, amount)
	return nil
}

// Build build the final value transaction: not signed and without data payload

func (txb *Builder) Build(useAllInputs bool) *valuetransaction.Transaction {
	if txb.finalized {
		panic("using finalized transaction builder")
	}
	defer func() {
		txb.finalized = true
	}()
	if !useAllInputs {
		// filter out unconsumed inputs
		finp := txb.inputBalances[:0]
		for i := range txb.inputBalances {
			if len(txb.inputBalances[i].consumed) == 0 {
				continue
			}
			finp = append(finp, txb.inputBalances[i])
		}
		txb.inputBalances = finp
	}

	for i := range txb.inputBalances {
		for _, bal := range txb.inputBalances[i].remain {
			if bal.Value > 0 {
				txb.addToOutputs(txb.inputBalances[i].outputId.Address(), bal.Color, bal.Value)
				//bal.Value = 0
			}
		}
	}
	inps := make([]valuetransaction.OutputID, len(txb.inputBalances))
	for i := range inps {
		inps[i] = txb.inputBalances[i].outputId
	}
	sort.Slice(inps, func(i, j int) bool {
		return bytes.Compare(inps[i][:], inps[j][:]) < 0
	})
	outmap := make(map[address.Address][]*balance.Balance)
	for addr, balmap := range txb.outputBalances {
		outmap[addr] = make([]*balance.Balance, 0, len(balmap))
		for col, b := range balmap {
			if b <= 0 {
				panic("internal inconsistency: balance value must be positive")
			}
			outmap[addr] = append(outmap[addr], balance.New(col, b))
		}
		sort.Slice(outmap[addr], func(i, j int) bool {
			return bytes.Compare(outmap[addr][i].Color[:], outmap[addr][j].Color[:]) < 0
		})
	}
	return valuetransaction.New(
		valuetransaction.NewInputs(inps...),
		valuetransaction.NewOutputs(outmap),
	)
}

func (txb *Builder) Dump() string {
	ret := "inputs:\n"
	// remain
	for i := range txb.inputBalances {
		ret += txb.inputBalances[i].outputId.Address().String() + " - " +
			txb.inputBalances[i].outputId.TransactionID().String() + "\n"
		for _, bal := range txb.inputBalances[i].remain {
			ret += fmt.Sprintf("      remain %d %s\n", bal.Value, bal.Color.String())
		}
		for _, bal := range txb.inputBalances[i].consumed {
			ret += fmt.Sprintf("      consumed %d %s\n", bal.Value, bal.Color.String())
		}
	}
	ret += "outputs:\n"
	for addr, balmap := range txb.outputBalances {
		ret += fmt.Sprintf("        %s\n", addr.String())
		for c, b := range balmap {
			ret += fmt.Sprintf("                         %s: %d\n", c.String(), b)
		}
	}
	return ret
}
