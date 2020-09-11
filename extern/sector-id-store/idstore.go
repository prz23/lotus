package idstore

import (
	_ "bytes"
	"encoding/json"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/ipfs/go-datastore"
)
var StorageCounterDSPrefix = "/storage/nextid"
var AllIpPrefix = "AllIp"

type SectorId uint64
type SlaveIP string

type SectorIpRecord interface {
	Insert(id SectorId,ip SlaveIP) error
	Remove(id SectorId) error
	FindAll(k []SectorId) ([]SlaveIP,error)
	FindAllSort(k []SectorId) (map[SlaveIP][]SectorId,error)
	GetAllIp() ([]string,error)
}

//map[SectorId]SlaveIP

type SectorIdStore struct {
	ds   datastore.Datastore
	idip datastore.Key
	btf  datastore.Key
	allip  datastore.Key
}

func StartIdIpStore(ds dtypes.MetadataDS) SectorIpRecord {
	return NewIdIpStore(ds,datastore.NewKey("IpIdStore"),datastore.NewKey(StorageCounterDSPrefix),
		datastore.NewKey(AllIpPrefix))
}

func NewIdIpStore(ds dtypes.MetadataDS, name datastore.Key, name2 datastore.Key, name3 datastore.Key) *SectorIdStore {
	has, _ := ds.Has(name)
	if has {
		return &SectorIdStore{ds: ds, idip: name, btf:name2, allip: name3}
	}
//
	mapa := make(map[SectorId]SlaveIP)
	buf , _ := json.Marshal(mapa)
	size := len(buf)
	_ = ds.Put(name, buf[:size])
//
	ipcollections := make([]string,1)
	buf2 , _ := json.Marshal(ipcollections)
	size2 := len(buf2)
	_ = ds.Put(name3, buf2[:size2])
	return &SectorIdStore{ds: ds, idip: name, btf:name2, allip: name3}
}

func (sc *SectorIdStore) Insert(id SectorId,ip SlaveIP) error{
	has, err := sc.ds.Has(sc.idip)
	if err != nil {
		return  err
	}
	if has {
		curBytes, err := sc.ds.Get(sc.idip)
		if err != nil {
			return  err
		}
		var data map[SectorId]SlaveIP
		err = json.Unmarshal(curBytes,&data)
		if err != nil {
			return err
		}

		data[id] = ip
        _ = sc.saveip(string(ip))

		buf ,err := json.Marshal(data)
		if err != nil {
			return err
		}
		size := len(buf)

		return sc.ds.Put(sc.idip, buf[:size])
	}
	return nil
}

func (sc *SectorIdStore) Remove(id SectorId) error{
	has, err := sc.ds.Has(sc.idip)
	if err != nil {
		return  err
	}
	if has {
		curBytes, err := sc.ds.Get(sc.idip)
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

		return sc.ds.Put(sc.idip, buf[:size])
	}
	return nil
}

func (sc *SectorIdStore) FindAll(k []SectorId) ([]SlaveIP,error){
	has, err := sc.ds.Has(sc.idip)
	if err != nil {
		return  nil,err
	}
	if has {
		curBytes, err := sc.ds.Get(sc.idip)
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
	has, err := sc.ds.Has(sc.idip)
	if err != nil {
		return  nil,err
	}
	if has {
		curBytes, err := sc.ds.Get(sc.idip)
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

const base = 2000000

func (sc *SectorIdStore)saveip(ip string) error{
	has, err := sc.ds.Has(sc.allip)
	if err != nil {
		return  err
	}
	if has {
		curBytes, err := sc.ds.Get(sc.allip)
		if err != nil {
			return err
		}
		var data []string
		err = json.Unmarshal(curBytes,&data)
		if err != nil {
			return err
		}
		data = append(data,ip)

		bytes,err := json.Marshal(data)
		if err != nil {
			return err
		}

		size := len(bytes)

		return sc.ds.Put(sc.allip, bytes[:size])
	}
		return err
}

func (sc *SectorIdStore)GetAllIp() ([]string,error){
	has, err := sc.ds.Has(sc.allip)
	if err != nil {
		return  nil,err
	}
	if has {
		curBytes, err := sc.ds.Get(sc.allip)
		if err != nil {
			return nil,err
		}
		var data []string
		err = json.Unmarshal(curBytes,&data)
		if err != nil {
			return nil,err
		}

		return data,nil
	}
	return nil,err
}