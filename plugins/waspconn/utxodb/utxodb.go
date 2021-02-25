package utxodb

import (
	"errors"
	"github.com/iotaledger/goshimmer/packages/ledgerstate"
	"golang.org/x/xerrors"
	"sync"
)

// UtxoDB is the structure which contains all UTXODB transactions and ledger
type UtxoDB struct {
	transactions map[ledgerstate.TransactionID]*ledgerstate.Transaction
	utxo         map[ledgerstate.OutputID]ledgerstate.Output
	mutex        *sync.RWMutex
	genesisTxId  ledgerstate.TransactionID
}

// New creates new UTXODB instance
func New() *UtxoDB {
	u := &UtxoDB{
		transactions: make(map[ledgerstate.TransactionID]*ledgerstate.Transaction),
		utxo:         make(map[ledgerstate.OutputID]ledgerstate.Output),
		mutex:        &sync.RWMutex{},
	}
	u.genesisInit()
	return u
}

// ValidateTransaction check is the transaction can be added to the ledger
func (u *UtxoDB) ValidateTransaction(tx *ledgerstate.Transaction) error {
	if err := u.CheckInputsOutputs(tx); err != nil {
		return xerrors.Errorf("utxodb: %v: txid %s", err, tx.ID().String())
	}
	if !tx.SignaturesValid() {
		return xerrors.Errorf("utxodb: invalid signature txid = %s", tx.ID().String())
	}
	return nil
}

// AreConflicting checks if two transactions double-spend (has equal inputs)
func AreConflicting(tx1, tx2 *ledgerstate.Transaction) bool {
	if tx1.ID() == tx2.ID() {
		return true
	}
	for _, inp1 := range tx1.Essence().Inputs() {
		for _, inp2 := range tx2.Essence().Inputs() {
			if inp2.Compare(inp1) == 0 {
				return true
			}
		}
	}
	return false
}

// IsConfirmed checks if the transaction is in the UTXODB (in the ledger)
func (u *UtxoDB) IsConfirmed(txid *ledgerstate.TransactionID) bool {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	_, ok := u.transactions[*txid]
	return ok
}

func (u *UtxoDB) CheckCanBeAdded(tx *ledgerstate.Transaction) error {
	if _, ok := u.transactions[tx.ID()]; ok {
		return xerrors.Errorf("utxodb: duplicate transaction %s", tx.ID().String())
	}
	// check if outputs exist
	for _, inp := range tx.Essence().Inputs() {
		if inp.Type() != ledgerstate.UTXOInputType {
			return xerrors.Errorf("utxodb.AddTransaction: UTXOInputType expected")
		}
		utxoInp := inp.(*ledgerstate.UTXOInput)
		if _, ok := u.utxo[utxoInp.ReferencedOutputID()]; !ok {
			return xerrors.Errorf("utxodb.AddTransaction: referenced output not found. May be already spent. txid = %s", tx.ID().String())
		}
	}
	return nil
}

// AddTransaction adds transaction to UTXODB or return an error.
// The function ensures consistency of the UTXODB ledger
func (u *UtxoDB) AddTransaction(tx *ledgerstate.Transaction) error {
	if err := u.ValidateTransaction(tx); err != nil {
		return err
	}

	u.mutex.Lock()
	defer u.mutex.Unlock()

	if err := u.CheckCanBeAdded(tx); err != nil {
		return err
	}
	// delete consumed (referenced) outputs from ledger
	for _, inp := range tx.Essence().Inputs() {
		utxoInp := inp.(*ledgerstate.UTXOInput)
		delete(u.utxo, utxoInp.ReferencedOutputID())
	}
	// add outputs to the ledger
	for _, out := range tx.Essence().Outputs() {
		if out.ID().TransactionID() != tx.ID() {
			panic("utxodb.AddTransaction: incorrect output ID")
		}
		switch to := out.Clone().(type) {
		case *ledgerstate.SigLockedColoredOutput:
			to.UpdateMintingColor()
		case *ledgerstate.SigLockedSingleOutput:
		default:
			panic("utxodb.AddTransaction: unknown type")
		}
		u.utxo[out.ID()] = out
	}
	u.transactions[tx.ID()] = tx
	u.checkLedgerBalance()
	return nil
}

// GetTransaction retrieves value transaction by its hash (ID)
func (u *UtxoDB) GetTransaction(id ledgerstate.TransactionID) (*ledgerstate.Transaction, bool) {
	u.mutex.RLock()
	defer u.mutex.RUnlock()

	return u.getTransaction(id)
}

func (u *UtxoDB) getTransaction(id ledgerstate.TransactionID) (*ledgerstate.Transaction, bool) {
	tx, ok := u.transactions[id]
	return tx, ok
}

func (u *UtxoDB) mustGetTransaction(id ledgerstate.TransactionID) *ledgerstate.Transaction {
	tx, ok := u.transactions[id]
	if !ok {
		panic(xerrors.Errorf("utxodb.mustGetTransaction: tx id doesn't exist: %s", id.String()))
	}
	return tx
}

// MustGetTransaction same as GetTransaction only panics if transaction is not in UTXODB
func (u *UtxoDB) MustGetTransaction(id ledgerstate.TransactionID) *ledgerstate.Transaction {
	u.mutex.RLock()
	defer u.mutex.RUnlock()
	return u.mustGetTransaction(id)
}

// GetAddressOutputs returns outputs contained in the address
func (u *UtxoDB) GetAddressOutputs(addr ledgerstate.Address) ledgerstate.Outputs {
	u.mutex.RLock()
	defer u.mutex.RUnlock()

	return u.getAddressOutputs(addr)
}

func (u *UtxoDB) getAddressOutputs(addr ledgerstate.Address) ledgerstate.Outputs {
	outs := make([]ledgerstate.Output, 0)
	for _, out := range u.utxo {
		if out.Address() == addr {
			outs = append(outs, out)
		}
	}
	return ledgerstate.NewOutputs(outs...)
}

func (u *UtxoDB) getOutputTotal(outid ledgerstate.OutputID) (uint64, error) {
	out, ok := u.utxo[outid]
	if !ok {
		return 0, xerrors.Errorf("no such output: %s", outid.String())
	}
	ret := uint64(0)
	out.Balances().ForEach(func(_ ledgerstate.Color, bal uint64) bool {
		ret += bal
		return true
	})
	return ret, nil
}

func (u *UtxoDB) checkLedgerBalance() {
	total := uint64(0)
	for outp := range u.utxo {
		b, err := u.getOutputTotal(outp)
		if err != nil {
			panic("utxodb: wrong ledger balance: " + err.Error())
		}
		total += b
	}
	if total != supply {
		panic("utxodb: wrong ledger balance")
	}
}
