package umrpc

import (
	"sync"
)

type SectorId = uint64
type SlaveIP = string

type State struct {
	IdIpMap map[SectorId]SlaveIP
}

func newMap() State {
	IdIpMap2 := make(map[SectorId]SlaveIP)
	state := State{
		IdIpMap2,
	}
	return state
}

func (s *singleton) Insert(k SectorId, v SlaveIP) {
	s.IdIpMap[k] = v
}

func (s *singleton) Find(k SectorId) SlaveIP {
	return s.IdIpMap[k]
}

func (s *singleton) FindAll(k []SectorId) []SlaveIP {
	lenInput := len(k)

	SliceSlaveIP := make([]SlaveIP, lenInput)

	for _, b := range k {
		v := s.IdIpMap[b]
		SliceSlaveIP = append(SliceSlaveIP, v)
	}

	return SliceSlaveIP
}

func (s *singleton) FindAllSort(k []SectorId) map[SlaveIP][]SectorId {

	SortedMap := make(map[SlaveIP][]SectorId)

	for _, sectorid := range k {
		v := s.IdIpMap[sectorid]
		origin := SortedMap[v]
		SortedMap[v] = append(origin, sectorid)
	}

	return SortedMap
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
