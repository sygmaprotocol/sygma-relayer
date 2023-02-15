// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"fmt"
	"sync"

	"github.com/ChainSafe/sygma-relayer/chains/substrate/events"
	"github.com/centrifuge/go-substrate-rpc-client/v4/client"
	"github.com/centrifuge/go-substrate-rpc-client/v4/rpc/chain"

	"github.com/centrifuge/go-substrate-rpc-client/v4/rpc"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type Connection struct {
	chain.Chain
	*rpc.RPC
	meta        types.Metadata // Latest chain metadata
	metaLock    sync.RWMutex   // Lock metadata for updates, allows concurrent reads
	GenesisHash types.Hash     // Chain genesis hash
}

func NewSubstrateConnection(url string) (*Connection, error) {
	c := &Connection{}
	client, err := client.Connect(url)
	if err != nil {
		return nil, err
	}
	rpc, err := rpc.NewRPC(client)
	if err != nil {
		return nil, err
	}
	c.RPC = rpc
	c.Chain = rpc.Chain

	// Fetch metadata
	meta, err := c.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, err
	}
	c.meta = *meta
	// Fetch genesis hash
	genesisHash, err := c.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return nil, err
	}
	c.GenesisHash = genesisHash
	return c, nil
}

func (c *Connection) GetMetadata() (meta types.Metadata) {
	c.metaLock.RLock()
	meta = c.meta
	c.metaLock.RUnlock()
	return meta
}

func (c *Connection) UpdateMetatdata() error {
	c.metaLock.Lock()
	meta, err := c.RPC.State.GetMetadataLatest()
	if err != nil {
		c.metaLock.Unlock()
		return err
	}
	c.meta = *meta
	c.metaLock.Unlock()
	return nil
}

func (c *Connection) GetBlockEvents(hash types.Hash) (*events.Events, error) {
	fmt.Println(hash.Hex())
	meta := c.GetMetadata()
	key, err := types.CreateStorageKey(&meta, "System", "Events", nil)
	if err != nil {
		return nil, err
	}

	var raw types.EventRecordsRaw
	_, err = c.RPC.State.GetStorage(key, &raw, hash)
	if err != nil {
		return nil, err
	}
	evts := &events.Events{}
	err = raw.DecodeEventRecords(&meta, evts)
	if err != nil {
		return nil, err
	}
	return evts, nil
}

/* func DecodeEventRecords(m *types.Metadata, t interface{}, e types.EventRecordsRaw) error { //nolint:funlen
fmt.Println(fmt.Sprintf("will decode event records from raw hex: %#x", e))
fmt.Println("sa\n\n\n\n\nfdlkdshfjshfkjdsfkjdsbfdsfds")
// ensure t is a pointer
ttyp := reflect.TypeOf(t)
if ttyp.Kind() != reflect.Ptr {
	return errors.New("target must be a pointer, but is " + fmt.Sprint(ttyp))
}
// ensure t is not a nil pointer
tval := reflect.ValueOf(t)
if tval.IsNil() {
	return errors.New("target is a nil pointer")
}
val := tval.Elem()
typ := val.Type()
// ensure val can be set
if !val.CanSet() {
	return fmt.Errorf("unsettable value %v", typ)
}
// ensure val points to a struct
if val.Kind() != reflect.Struct {
	return fmt.Errorf("target must point to a struct, but is " + fmt.Sprint(typ))
}

decoder := scale.NewDecoder(bytes.NewReader(e))

// determine number of events
n, err := decoder.DecodeUintCompact()
if err != nil {
	return err
}

fmt.Println(fmt.Sprintf("found %v events", n))

// iterate over events
for i := uint64(0); i < n.Uint64(); i++ {
	fmt.Println(fmt.Sprintf("decoding event #%v", i))

	// decode Phase
	phase := types.Phase{}
	err := decoder.Decode(&phase)
	if err != nil {
		return fmt.Errorf("unable to decode Phase for event #%v: %v", i, err)
	}

	// decode EventID
	id := types.EventID{}
	err = decoder.Decode(&id)
	if err != nil {
		return fmt.Errorf("unable to decode EventID for event #%v: %v", i, err)
	}

	fmt.Println(fmt.Sprintf("event #%v has EventID %v", i, id))

	// ask metadata for method & event name for event
	moduleName, eventName, err := m.FindEventNamesForEventID(id)
	// moduleName, eventName, err := "System", "ExtrinsicSuccess", nil
	if err != nil {
		return fmt.Errorf("unable to find event with EventID %v in metadata for event #%v: %s", id, i, err)
	}

	fmt.Println(fmt.Sprintf("event #%v is in module %v with event name %v", i, moduleName, eventName))

	// check whether name for eventID exists in t
	field := val.FieldByName(fmt.Sprintf("%v_%v", moduleName, eventName))
	if !field.IsValid() {
		return fmt.Errorf("unable to find field %v_%v for event #%v with EventID %v", moduleName, eventName, i, id)
	}

	// create a pointer to with the correct type that will hold the decoded event
	holder := reflect.New(field.Type().Elem())

	// ensure first field is for Phase, last field is for Topics
	numFields := holder.Elem().NumField()
	if numFields < 2 {
		return fmt.Errorf("expected event #%v with EventID %v, field %v_%v to have at least 2 fields "+
			"(for Phase and Topics), but has %v fields", i, id, moduleName, eventName, numFields)
	}
	phaseField := holder.Elem().FieldByIndex([]int{0})
	if phaseField.Type() != reflect.TypeOf(phase) {
		return fmt.Errorf("expected the first field of event #%v with EventID %v, field %v_%v to be of type "+
			"types.Phase, but got %v", i, id, moduleName, eventName, phaseField.Type())
	}
	topicsField := holder.Elem().FieldByIndex([]int{numFields - 1})
	if topicsField.Type() != reflect.TypeOf([]types.Hash{}) {
		return fmt.Errorf("expected the last field of event #%v with EventID %v, field %v_%v to be of type "+
			"[]types.Hash for Topics, but got %v", i, id, moduleName, eventName, topicsField.Type())
	}

	// set the phase we decoded earlier
	phaseField.Set(reflect.ValueOf(phase))

	// set the remaining fields
	for j := 1; j < numFields; j++ {
		if moduleName == "SygmaBridge" && eventName == "Deposit" && j == 9 {

			fmt.Println("cudedsasdasdasda\n")
			/* 	fmt.Println(holder.Elem().FieldByIndex([]int{j - 1}))
			err = decoder.Decode([holder.Elem().FieldByIndex([]int{j - 1})]byte) */
/* 				if err != nil {
					return fmt.Errorf("unable to decode field %v event #%v with EventID %v, field %v_%v: %v", j, i, id, moduleName,
						eventName, err)
				} else {
					err = decoder.Decode(holder.Elem().FieldByIndex([]int{j}).Addr().Interface())
					if err != nil {
						return fmt.Errorf("unable to decode field %v event #%v with EventID %v, field %v_%v: %v", j, i, id, moduleName,
							eventName, err)

					}
				}
			}
			// add the decoded event to the slice
			field.Set(reflect.Append(field, holder.Elem()))

			fmt.Println(fmt.Sprintf("decoded event #%v", i))
		}
		return nil
	}
	return nil
}  */
