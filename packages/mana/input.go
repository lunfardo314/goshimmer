package mana

import (
	"sync"

	"github.com/golang/protobuf/proto"

	"github.com/iotaledger/goshimmer/packages/errors"
	manaproto "github.com/iotaledger/goshimmer/packages/mana/proto"
)

type Input struct {
	coinAmount        uint64
	coinAmountMutex   sync.RWMutex
	receivedTime      uint64
	receivedTimeMutex sync.RWMutex
}

func NewInput(coinAmount uint64, receivedTime uint64) *Input {
	return &Input{
		coinAmount:   coinAmount,
		receivedTime: receivedTime,
	}
}

func (input *Input) GetCoinAmount() uint64 {
	input.coinAmountMutex.RLock()
	defer input.coinAmountMutex.RUnlock()

	return input.coinAmount
}

func (input *Input) SetCoinAmount(coinAmount uint64) {
	input.coinAmountMutex.Lock()
	defer input.coinAmountMutex.Unlock()

	input.coinAmount = coinAmount
}

func (input *Input) GetReceivedTime() uint64 {
	input.receivedTimeMutex.RLock()
	defer input.receivedTimeMutex.RUnlock()

	return input.receivedTime
}

func (input *Input) SetReceivedTime(receivedTime uint64) {
	input.receivedTimeMutex.Lock()
	defer input.receivedTimeMutex.Unlock()

	input.receivedTime = receivedTime
}

func (input *Input) ToProto() (result *manaproto.Input) {
	input.receivedTimeMutex.RLock()
	input.coinAmountMutex.RLock()
	defer input.receivedTimeMutex.RUnlock()
	defer input.coinAmountMutex.RUnlock()

	return &manaproto.Input{
		CoinAmount:   input.coinAmount,
		ReceivedTime: input.receivedTime,
	}
}

func (input *Input) FromProto(proto *manaproto.Input) {
	input.receivedTimeMutex.Lock()
	input.coinAmountMutex.Lock()
	defer input.receivedTimeMutex.Unlock()
	defer input.coinAmountMutex.Unlock()

	input.coinAmount = proto.CoinAmount
	input.receivedTime = proto.ReceivedTime
}

func (input *Input) MarshalBinary() (result []byte, err errors.IdentifiableError) {
	if marshaledData, marshalErr := proto.Marshal(input.ToProto()); marshalErr != nil {
		err = ErrMarshalFailed.Derive(marshalErr, "marshal failed")
	} else {
		result = marshaledData
	}

	return
}

func (input *Input) UnmarshalBinary(data []byte) (err errors.IdentifiableError) {
	var unmarshaledProto manaproto.Input
	if unmarshalError := proto.Unmarshal(data, &unmarshaledProto); unmarshalError != nil {
		err = ErrUnmarshalFailed.Derive(unmarshalError, "unmarshal failed")
	} else {
		input.FromProto(&unmarshaledProto)
	}

	return
}
