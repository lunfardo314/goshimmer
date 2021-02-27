package txbuilder

import (
	"github.com/iotaledger/goshimmer/packages/ledgerstate"
	"github.com/iotaledger/hive.go/identity"
	"golang.org/x/xerrors"
	"time"
)

type Builder struct {
	version           ledgerstate.TransactionEssenceVersion
	accessPledgeID    identity.ID
	consensusPledgeID identity.ID
	timestamp         time.Time
	consumables       []*ConsumableOutput
	senderAddress     ledgerstate.Address
	doNotCompress     bool
}

func New(outputs ledgerstate.Outputs) (*Builder, error) {
	senderAddr, err := takeSenderAddress(outputs)
	if err != nil {
		return nil, err
	}
	ret := &Builder{
		timestamp:     time.Now(),
		consumables:   make([]*ConsumableOutput, len(outputs)),
		senderAddress: senderAddr,
	}
	for i, out := range outputs {
		ret.consumables[i] = NewConsumableOutput(out)
	}
	return ret, nil
}

func MustNew(outputs ledgerstate.Outputs) *Builder {
	ret, err := New(outputs)
	if err != nil {
		panic(err)
	}
	return ret
}

func takeSenderAddress(outputs ledgerstate.Outputs) (ledgerstate.Address, error) {
	var ret ledgerstate.Address
	for _, out := range outputs {
		if ret != nil && ret.Array() != out.Address().Array() {
			return nil, xerrors.New("txbuilder.takeSenderAddress: all outputs must be from the same address")
		}
		ret = out.Address()
	}
	return ret, nil
}

func (b *Builder) WithVersion(v ledgerstate.TransactionEssenceVersion) *Builder {
	b.version = v
	return b
}

func (b *Builder) WithTime(t time.Time) *Builder {
	b.timestamp = t
	return b
}

func (b *Builder) WithAccessPledge(id identity.ID) *Builder {
	b.accessPledgeID = id
	return b
}

func (b *Builder) WithConsensusPledge(id identity.ID) *Builder {
	b.consensusPledgeID = id
	return b
}

func (b *Builder) WithOutputCompression(compress bool) *Builder {
	b.doNotCompress = !compress
	return b
}

func (b *Builder) BuildIOTATransfer(targetAddress ledgerstate.Address, amount uint64) (*ledgerstate.TransactionEssence, error) {
	if err := ConsumeIOTA(amount, b.consumables...); err != nil {
		return nil, err
	}
	outputs := ledgerstate.Outputs{ledgerstate.NewSigLockedSingleOutput(amount, targetAddress)}
	inputConsumables := b.consumables
	if !b.doNotCompress {
		inputConsumables = SelectConsumed(b.consumables...)
	}
	reminderBalances := make(map[ledgerstate.Color]uint64)
	ConsumeRemaining(reminderBalances, inputConsumables...)
	if len(reminderBalances) != 0 {
		if numIotas, ok := reminderBalances[ledgerstate.ColorIOTA]; ok && len(reminderBalances) == 1 {
			outputs = append(outputs, ledgerstate.NewSigLockedSingleOutput(numIotas, b.senderAddress))
		} else {
			bals := ledgerstate.NewColoredBalances(reminderBalances)
			outputs = append(outputs, ledgerstate.NewSigLockedColoredOutput(bals, b.senderAddress))
		}
	}
	inputs := MakeUTXOInputs(inputConsumables...)
	return ledgerstate.NewTransactionEssence(b.version, b.timestamp, b.accessPledgeID, b.consensusPledgeID, inputs, outputs), nil
}
