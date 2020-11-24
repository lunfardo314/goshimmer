package mana

import (
	"math"
	"testing"
	"time"

	"github.com/iotaledger/hive.go/identity"
	"github.com/iotaledger/hive.go/marshalutil"
	"github.com/stretchr/testify/assert"
)

func TestPersistableBaseMana_Bytes(t *testing.T) {
	p := newPersistableMana()
	marshalUtil := marshalutil.New()
	marshalUtil.WriteInt64(int64(p.ManaType))
	marshalUtil.WriteUint16(uint16(len(p.BaseValues)))
	for _, val := range p.BaseValues {
		marshalUtil.WriteUint64(math.Float64bits(val))
	}
	marshalUtil.WriteUint16(uint16(len(p.EffectiveValues)))
	for _, val := range p.EffectiveValues {
		marshalUtil.WriteUint64(math.Float64bits(val))
	}

	marshalUtil.WriteTime(p.LastUpdated)
	marshalUtil.WriteBytes(p.NodeID.Bytes())

	bytes := marshalUtil.Bytes()
	assert.Equal(t, bytes, p.Bytes(), "should be equal")

}
func TestPersistableBaseMana_ObjectStorageKey(t *testing.T) {
	p := newPersistableMana()
	key := p.ObjectStorageKey()
	assert.Equal(t, identity.ID{}.Bytes(), key, "should be equal")
}

func TestPersistableBaseMana_ObjectStorageValue(t *testing.T) {
	p := newPersistableMana()
	val := p.ObjectStorageValue()
	assert.Equal(t, p.Bytes(), val, "should be equal")
}

func TestPersistableBaseMana_Update(t *testing.T) {
	p := newPersistableMana()
	assert.Panics(t, func() {
		p.Update(nil)
	}, "should have paniced")
}

func TestPersistableBaseMana_FromBytes(t *testing.T) {
	p1 := newPersistableMana()
	p2, _, err := FromBytes(p1.Bytes())
	assert.Nil(t, err, "should not have returned error")
	assert.Equal(t, p1.Bytes(), p2.Bytes(), "should be equal")
}

func TestFromStorageKey(t *testing.T) {
	p := newPersistableMana()
	p1, err := FromObjectStorage(p.NodeID.Bytes(), p.Bytes())
	assert.Nil(t, err, "should not have returned error")
	assert.Equal(t, p.NodeID, p1.(*PersistableBaseMana).NodeID, "should be equal")
}

func newPersistableMana() *PersistableBaseMana {
	return &PersistableBaseMana{
		ManaType:        ConsensusMana,
		BaseValues:      []float64{1, 1},
		EffectiveValues: []float64{1, 1},
		LastUpdated:     time.Now(),
		NodeID:          identity.ID{},
	}
}
