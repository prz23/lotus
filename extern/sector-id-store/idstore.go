package idstore

import (
	"encoding/json"
	"github.com/ipfs/go-datastore"
)

type SectorId uint64
type SlaveIP string

type SectorIpRecord interface {
	Insert(id SectorId,ip SlaveIP) error
	Remove(id SectorId) error
	FindAll(k []SectorId) ([]SlaveIP,error)
	FindAllSort(k []SectorId) (map[SlaveIP][]SectorId,error)
}

//map[SectorId]SlaveIP

type SectorIdStore struct {
	ds   datastore.Datastore
	name datastore.Key
}

func StartIdIpStore(ds datastore.Datastore) SectorIpRecord {
	return NewIdIpStore(ds,datastore.NewKey("IpIdStore"))
}

func NewIdIpStore(ds datastore.Datastore, name datastore.Key) *SectorIdStore {
	has, _ := ds.Has(name)
	if has {
		return &SectorIdStore{ds: ds, name: name}
	}

	mapa := make(map[SectorId]SlaveIP)
	buf , _ := json.Marshal(mapa)

	size := len(buf)
	_ = ds.Put(name, buf[:size])

	return &SectorIdStore{ds: ds, name: name}
}

func (sc *SectorIdStore) Insert(id SectorId,ip SlaveIP) error{
	has, err := sc.ds.Has(sc.name)
	if err != nil {
		return  err
	}
	if has {
		curBytes, err := sc.ds.Get(sc.name)
		if err != nil {
			return  err
		}
		var data map[SectorId]SlaveIP
		err = json.Unmarshal(curBytes,&data)
		if err != nil {
			return err
		}

		data[id] = ip

		buf ,err := json.Marshal(data)
		if err != nil {
			return err
		}
		size := len(buf)

		return sc.ds.Put(sc.name, buf[:size])
	}
	return nil
}

func (sc *SectorIdStore) Remove(id SectorId) error{
	has, err := sc.ds.Has(sc.name)
	if err != nil {
		return  err
	}
	if has {
		curBytes, err := sc.ds.Get(sc.name)
		if err != nil {
			return  err
		}
		var data map[SectorId]SlaveIP
		err = json.Unmarshal(curBytes,&data)
		if err != nil {
			return err
		}

		delete(data,id)

		buf ,err := json.Marshal(data)
		if err != nil {
			return err
		}
		size := len(buf)

		return sc.ds.Put(sc.name, buf[:size])
	}
	return nil
}

func (sc *SectorIdStore) FindAll(k []SectorId) ([]SlaveIP,error){
	has, err := sc.ds.Has(sc.name)
	if err != nil {
		return  nil,err
	}
	if has {
		curBytes, err := sc.ds.Get(sc.name)
		if err != nil {
			return  nil,err
		}
		var data map[SectorId]SlaveIP
		err = json.Unmarshal(curBytes,&data)
		if err != nil {
			return nil,err
		}

		lenInput := len(k)
		SliceSlaveIP := make([]SlaveIP, lenInput)

		for _, b := range k {
			v := data[b]
			SliceSlaveIP = append(SliceSlaveIP, v)
		}

		return SliceSlaveIP,nil
	}
	return nil,nil
}

func (sc *SectorIdStore) FindAllSort(k []SectorId) (map[SlaveIP][]SectorId,error){
	has, err := sc.ds.Has(sc.name)
	if err != nil {
		return  nil,err
	}
	if has {
		curBytes, err := sc.ds.Get(sc.name)
		if err != nil {
			return  nil,err
		}
		var data map[SectorId]SlaveIP
		err = json.Unmarshal(curBytes,&data)
		if err != nil {
			return nil,err
		}

		SortedMap := make(map[SlaveIP][]SectorId)

		for _, sectorid := range k {
			v := data[sectorid]
			origin := SortedMap[v]
			SortedMap[v] = append(origin, sectorid)
		}

		return SortedMap,nil
	}
	return nil,nil
}