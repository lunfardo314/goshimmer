package utxodb

import (
	"github.com/iotaledger/goshimmer/packages/ledgerstate"
	"github.com/iotaledger/hive.go/crypto/ed25519"
	"github.com/iotaledger/hive.go/identity"
	"golang.org/x/xerrors"
	"sync"
	"time"
)

const (
	supply          = uint64(100 * 1000 * 1000)
	genesisIndex    = 31415
	manaPledgeIndex = 271828
)

var (
	seed           = ed25519.NewSeed([]byte("EFonzaUz5ngYeDxbRKu8qV5aoSogUQ5qVSTSjn7hJ8FQ"))
	genesisKeyPair = NewKeyPairFromSeed(genesisIndex)
	essenceVersion = ledgerstate.TransactionEssenceVersion(0)
)

// UtxoDB is the structure which contains all UTXODB transactions and ledger
type UtxoDB struct {
	genesisKeyPair *ed25519.KeyPair
	transactions   map[ledgerstate.TransactionID]*ledgerstate.Transaction
	utxo           map[ledgerstate.OutputID]ledgerstate.Output
	mutex          *sync.RWMutex
	genesisTxId    ledgerstate.TransactionID
}

// New creates new UTXODB instance
func New() *UtxoDB {
	u := &UtxoDB{
		genesisKeyPair: genesisKeyPair,
		transactions:   make(map[ledgerstate.TransactionID]*ledgerstate.Transaction),
		utxo:           make(map[ledgerstate.OutputID]ledgerstate.Output),
		mutex:          &sync.RWMutex{},
	}
	u.genesisInit()
	return u
}

func NewKeyPairFromSeed(index int) *ed25519.KeyPair {
	return seed.KeyPair(uint64(index))
}

func (u *UtxoDB) genesisInit() {
	// create genesis transaction
	inputs := ledgerstate.NewInputs(ledgerstate.NewUTXOInput(ledgerstate.NewOutputID(ledgerstate.TransactionID{}, 0)))
	output := ledgerstate.NewSigLockedSingleOutput(supply, u.GetGenesisAddress())
	outputs := ledgerstate.NewOutputs(output)
	essence := ledgerstate.NewTransactionEssence(essenceVersion, time.Now(), identity.ID{}, identity.ID{}, inputs, outputs)
	signature := ledgerstate.NewED25519Signature(u.genesisKeyPair.PublicKey, u.genesisKeyPair.PrivateKey.Sign(essence.Bytes()))
	unlockBlock := ledgerstate.NewSignatureUnlockBlock(signature)
	genesisTx := ledgerstate.NewTransaction(essence, ledgerstate.UnlockBlocks{unlockBlock})

	u.genesisTxId = genesisTx.ID()
	u.transactions[u.genesisTxId] = genesisTx
	u.utxo[output.ID()] = output.Clone()
}

// GetGenesisSigScheme return signature scheme used by creator of genesis
func (u *UtxoDB) GetGenesisKeyPair() *ed25519.KeyPair {
	return genesisKeyPair
}

// GetGenesisAddress return address of genesis
func (u *UtxoDB) GetGenesisAddress() ledgerstate.Address {
	return ledgerstate.NewED25519Address(genesisKeyPair.PublicKey)
}

const RequestFundsAmount = 1337 // same as Goshimmer Faucet

func (u *UtxoDB) mustRequestFundsTx(target ledgerstate.Address) *ledgerstate.Transaction {
	sourceOutputs := u.GetAddressOutputs(u.GetGenesisAddress())
	if len(sourceOutputs) != 1 {
		panic(xerrors.New("requestFundsTx: should be only one genesis output"))
	}
	out := sourceOutputs[0]
	b, ok := out.Balances().Get(ledgerstate.ColorIOTA)
	if !ok || b < RequestFundsAmount {
		panic(xerrors.New("requestFundsTx: not enough iotas in genesis!"))
	}
	inputs := ledgerstate.NewInputs(ledgerstate.NewUTXOInput(out.ID()))
	out1 := ledgerstate.NewSigLockedSingleOutput(RequestFundsAmount, target)
	out2 := ledgerstate.NewSigLockedSingleOutput(b-RequestFundsAmount, u.GetGenesisAddress())
	outputs := ledgerstate.NewOutputs(out1, out2)
	essence := ledgerstate.NewTransactionEssence(essenceVersion, time.Now(), identity.ID{}, identity.ID{}, inputs, outputs)
	signature := ledgerstate.NewED25519Signature(u.genesisKeyPair.PublicKey, u.genesisKeyPair.PrivateKey.Sign(essence.Bytes()))
	unlockBlock := ledgerstate.NewSignatureUnlockBlock(signature)
	ret := ledgerstate.NewTransaction(essence, ledgerstate.UnlockBlocks{unlockBlock})
	return ret
}

// RequestFunds implements faucet: it sends 1337 IOTA tokens from genesis to the given address.
func (u *UtxoDB) RequestFunds(target ledgerstate.Address) (*ledgerstate.Transaction, error) {
	tx := u.mustRequestFundsTx(target)
	return tx, u.AddTransaction(tx)
}
