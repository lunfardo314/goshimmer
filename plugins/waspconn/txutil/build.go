package txutil

import (
	"github.com/iotaledger/goshimmer/packages/ledgerstate"
	"github.com/iotaledger/hive.go/crypto/ed25519"
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
	compress          bool // default is do not compress all outputs to few, only take minimum for outputs
	outputs           []ledgerstate.Output
}

func NewBuilder(outputs ledgerstate.Outputs) *Builder {
	senderAddr, err := takeSenderAddress(outputs)
	if err != nil {
		return nil
	}
	ret := &Builder{
		timestamp:     time.Now(),
		consumables:   make([]*ConsumableOutput, len(outputs)),
		senderAddress: senderAddr,
		outputs:       make([]ledgerstate.Output, 0),
	}
	for i, out := range outputs {
		ret.consumables[i] = NewConsumableOutput(out)
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
	b.compress = compress
	return b
}

func (b *Builder) AddIOTATransfer(targetAddress ledgerstate.Address, amount uint64) error {
	if err := ConsumeIOTA(amount, b.consumables...); err != nil {
		return err
	}
	b.outputs = append(b.outputs, ledgerstate.NewSigLockedSingleOutput(amount, targetAddress))
	return nil

}

func (b *Builder) addReminderOutput() []*ConsumableOutput {
	inputConsumables := b.consumables
	if !b.compress {
		inputConsumables = SelectConsumed(b.consumables...)
	}
	reminderBalances := make(map[ledgerstate.Color]uint64)
	ConsumeRemaining(reminderBalances, inputConsumables...)
	if len(reminderBalances) != 0 {
		if numIotas, ok := reminderBalances[ledgerstate.ColorIOTA]; ok && len(reminderBalances) == 1 {
			b.outputs = append(b.outputs, ledgerstate.NewSigLockedSingleOutput(numIotas, b.senderAddress))
		} else {
			bals := ledgerstate.NewColoredBalances(reminderBalances)
			b.outputs = append(b.outputs, ledgerstate.NewSigLockedColoredOutput(bals, b.senderAddress))
		}
	}
	return inputConsumables
}

func (b *Builder) BuildEssence() *ledgerstate.TransactionEssence {
	inputConsumables := b.addReminderOutput()
	outputs := ledgerstate.NewOutputs(b.outputs...)
	inputs := MakeUTXOInputs(inputConsumables...)
	return ledgerstate.NewTransactionEssence(b.version, b.timestamp, b.accessPledgeID, b.consensusPledgeID, inputs, outputs)
}

func (b *Builder) BuildWithED25519(keyPair *ed25519.KeyPair) *ledgerstate.Transaction {
	essence := b.BuildEssence()
	data := essence.Bytes()
	signature := ledgerstate.NewED25519Signature(keyPair.PublicKey, keyPair.PrivateKey.Sign(data))
	if !signature.AddressSignatureValid(b.senderAddress, data) {
		panic("BuildWithED25519: internal error, signature invalid")
	}
	unlockBlocks := unlockBlocksFromSignature(signature, len(essence.Inputs()))
	return ledgerstate.NewTransaction(essence, unlockBlocks)
}

func unlockBlocksFromSignature(signature ledgerstate.Signature, n int) ledgerstate.UnlockBlocks {
	ret := make(ledgerstate.UnlockBlocks, n)
	ret[0] = ledgerstate.NewSignatureUnlockBlock(signature)
	for i := 1; i < n; i++ {
		ret[i] = ledgerstate.NewReferenceUnlockBlock(0)
	}
	return ret
}
