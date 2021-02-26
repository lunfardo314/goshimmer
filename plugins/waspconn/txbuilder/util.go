package txbuilder

import "github.com/iotaledger/goshimmer/packages/ledgerstate"

func EqualAddresses(addr1, addr2 ledgerstate.Address) bool {
	if addr1 == nil || addr2 == nil {
		return false
	}
	if addr1 == addr2 {
		return true
	}
	return addr1.Array() == addr2.Array()
}
