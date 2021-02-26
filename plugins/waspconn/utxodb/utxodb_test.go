package utxodb

import (
	"github.com/iotaledger/goshimmer/packages/ledgerstate"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBasic(t *testing.T) {
	u := New()
	genTx, ok := u.GetTransaction(u.genesisTxId)
	assert.Equal(t, ok, true)
	assert.Equal(t, genTx.ID(), u.genesisTxId)
}

func getBalance(u *UtxoDB, address ledgerstate.Address) uint64 {
	gout := u.GetAddressOutputs(address)
	total := uint64(0)
	for _, out := range gout {
		sum, err := u.getOutputTotal(out.ID())
		if err != nil {
			panic(err)
		}
		total += sum
	}
	return total
}

func TestGenesis(t *testing.T) {
	u := New()
	assert.Equal(t, supply, getBalance(u, u.GetGenesisAddress()))
	u.checkLedgerBalance()
}

//
//func TestRequestFunds(t *testing.T) {
//	u := New()
//	addr := NewSigScheme("C6hPhCS2E2dKUGS3qj4264itKXohwgL3Lm2fNxayAKr", 0).Address()
//	_, err := u.RequestFunds(addr)
//	assert.NoError(t, err)
//	assert.EqualValues(t, supply-RequestFundsAmount, getBalance(u, u.GetGenesisSigScheme().Address()))
//	assert.EqualValues(t, RequestFundsAmount, getBalance(u, addr))
//	u.checkLedgerBalance()
//}
//
//func TestTransferAndBook(t *testing.T) {
//	u := New()
//
//	addr := NewSigScheme("C6hPhCS2E2dKUGS3qj4264itKXohwgL3Lm2fNxayAKr", 0).Address()
//	tx, err := u.RequestFunds(addr)
//	assert.NoError(t, err)
//	assert.EqualValues(t, supply-RequestFundsAmount, getBalance(u, u.GetGenesisSigScheme().Address()))
//	assert.EqualValues(t, RequestFundsAmount, getBalance(u, addr))
//	u.checkLedgerBalance()
//
//	err = u.AddTransaction(tx)
//	assert.Error(t, err)
//}
