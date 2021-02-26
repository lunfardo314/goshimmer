package utxodb

import (
	"github.com/iotaledger/goshimmer/packages/ledgerstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.EqualValues(t, supply, getBalance(u, u.GetGenesisAddress()))
	u.checkLedgerBalance()
}

func TestRequestFunds(t *testing.T) {
	u := New()
	user := NewKeyPairFromSeed(2)
	addr := ledgerstate.NewED25519Address(user.PublicKey)
	_, err := u.RequestFunds(addr)
	require.NoError(t, err)
	require.EqualValues(t, supply-RequestFundsAmount, getBalance(u, u.GetGenesisAddress()))
	require.EqualValues(t, RequestFundsAmount, getBalance(u, addr))
	u.checkLedgerBalance()
}

func TestAddTransactionFail(t *testing.T) {
	u := New()
	user := NewKeyPairFromSeed(2)
	addr := ledgerstate.NewED25519Address(user.PublicKey)
	tx, err := u.RequestFunds(addr)
	require.NoError(t, err)
	require.EqualValues(t, supply-RequestFundsAmount, getBalance(u, u.GetGenesisAddress()))
	require.EqualValues(t, RequestFundsAmount, getBalance(u, addr))
	u.checkLedgerBalance()
	err = u.AddTransaction(tx)
	require.Error(t, err)
	u.checkLedgerBalance()
}
