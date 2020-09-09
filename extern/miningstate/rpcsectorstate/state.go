package sectorstate

import (
	"sync"
)

var mut sync.Mutex
var mut2 sync.Mutex

type SectorId = uint64
type SlaveIP = string

type State struct {
	IdIpMap map[SectorId]SlaveIP
	BusyIpMap map[SlaveIP]bool
}

func newMap() State {
	IdIpMap2 := make(map[SectorId]SlaveIP)
	BusyIpMap2 := make(map[SlaveIP]bool)
	state := State{
		IdIpMap2,
		BusyIpMap2,
	}
	return state
}

func (s *singleton) Insert(k SectorId, v SlaveIP) {
	mut.Lock()
	defer mut.Unlock()
	s.IdIpMap[k] = v
}

func (s *singleton) Find(k SectorId) SlaveIP {
	mut.Lock()
	defer mut.Unlock()
	return s.IdIpMap[k]
}

func (s *singleton) FindAll(k []SectorId) []SlaveIP {
	mut.Lock()
	defer mut.Unlock()

	lenInput := len(k)
	SliceSlaveIP := make([]SlaveIP, lenInput)

	for _, b := range k {
		v := s.IdIpMap[b]
		SliceSlaveIP = append(SliceSlaveIP, v)
	}

	return SliceSlaveIP
}

func (s *singleton) FindAllSort(k []SectorId) map[SlaveIP][]SectorId {

	mut.Lock()
	defer mut.Unlock()

	SortedMap := make(map[SlaveIP][]SectorId)

	for _, sectorid := range k {
		v := s.IdIpMap[sectorid]
		origin := SortedMap[v]
		SortedMap[v] = append(origin, sectorid)
	}

	return SortedMap
}

func (s *singleton) SetBusy(ip SlaveIP){
	mut2.Lock()
	defer mut2.Unlock()
	s.BusyIpMap[ip] = true
}

func (s *singleton) SetFree(ip SlaveIP){
	mut2.Lock()
	defer mut2.Unlock()
	s.BusyIpMap[ip] = false
}

func (s *singleton) GetFreeIP() SlaveIP{
	mut2.Lock()
	defer mut2.Unlock()
	for ip,busy := range s.BusyIpMap{
		if busy == false {
			return ip
		}
	}
	return ""
}

func (s *singleton) AddNewSlave(ip SlaveIP){
	mut2.Lock()
	defer mut2.Unlock()
	s.BusyIpMap[ip] = false
}

func (s *singleton) DeleteSlave(ip SlaveIP){
	mut2.Lock()
	defer mut2.Unlock()
	delete(s.BusyIpMap,ip)
}

func (s *singleton) HasRegister(ip SlaveIP) bool{
	_,ok := s.BusyIpMap[ip]
	return ok
}

type singleton struct {
	State
}

var ins *singleton
var once sync.Once

func GetIns() *singleton {
	once.Do(func() {
		s := newMap()
		ins = &singleton{s}
	})
	return ins
}
