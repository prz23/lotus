package filter

import (
	"encoding/json"
	"github.com/ipfs/go-datastore"
)

type LocalSectorRecord struct {
	ds   datastore.Datastore
	name datastore.Key
}

func New(ds datastore.Datastore, name datastore.Key) *LocalSectorRecord {
	has, _ := ds.Has(name)
	if has {
		return &LocalSectorRecord{ds: ds, name: name}
	}

	mapa := make(map[uint64]bool)
	buf , _ := json.Marshal(mapa)

	size := len(buf)
	_ = ds.Put(name, buf[:size])

	return &LocalSectorRecord{ds: ds, name: name}
}

func (sc *LocalSectorRecord) Insert(id uint64) error{

	has, err := sc.ds.Has(sc.name)
	if err != nil {
		return  err
	}
	if has {
		curBytes, err := sc.ds.Get(sc.name)
		if err != nil {
			return  err
		}
		var data map[uint64]bool
		err = json.Unmarshal(curBytes,&data)
		if err != nil {
			return err
		}

		data[id] = true

		buf ,err := json.Marshal(data)
		if err != nil {
			return err
		}
		size := len(buf)

		return sc.ds.Put(sc.name, buf[:size])

	}
	return nil
}

func (sc *LocalSectorRecord) Remove(id uint64) error{

	has, err := sc.ds.Has(sc.name)
	if err != nil {
		return  err
	}

	if has {
		curBytes, err := sc.ds.Get(sc.name)
		if err != nil {
			return  err
		}
		var data map[uint64]bool
		err = json.Unmarshal(curBytes,&data)
		if err != nil {
			return err
		}
		data[id] = false

		buf ,err := json.Marshal(data)
		if err != nil {
			return err
		}
		size := len(buf)

		return sc.ds.Put(sc.name, buf[:size])
	}
	return nil
}

func  (sc *LocalSectorRecord) Filter(selectedSectors []uint64) ([]uint64,error) {

	//has, err := sc.ds.Has(sc.name)
	//	//if err != nil {
	//	//	return  nil, err
	//	//}
	//	//log.Info("===========has==01")
	//	//if has {
	//	//	curBytes, err := sc.ds.Get(sc.name)
	//	//	if err != nil {
	//	//		return  nil, err
	//	//	}
	//	//	log.Info("===========has==01")
	//	//	var data []interface{}
	//	//	err = json.Unmarshal(curBytes,&data)
	//	//	if err != nil {
	//	//		return nil, err
	//	//	}
	//	//
	//	//	sectorset := set.NewSetFromSlice(data)
	//	//
	//	//	hasSet := make([]uint64,1,1)
	//	//	for _,x := range selectedSectors{
	//	//		if sectorset.Contains(x) {
	//	//			hasSet = append(hasSet,x)
	//	//		}
	//	//	}
	//	//
	//	//	return hasSet,nil
	//	//
	//	//}
	return nil, nil
}

func  (sc *LocalSectorRecord) Contains(selectedSectors uint64) (bool,error) {

	has, err := sc.ds.Has(sc.name)
	if err != nil {
		return  false, err
	}
	if has {
		curBytes, err := sc.ds.Get(sc.name)
		if err != nil {
			return  false,err
		}
		var data map[uint64]bool
		err = json.Unmarshal(curBytes,&data)
		if err != nil {
			return false,err
		}
		return data[selectedSectors],nil
	}
	return false, err
}