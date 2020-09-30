package sectorstate

import (
	"fmt"
	"testing"
	)

func TestA(t *testing.T) {

	s := GetIns()
	s.Insert(5, "192.168.0.1")
	ip := s.Find(5)
	fmt.Println("IP is ", ip)

	s.Insert(6, "192.168.31.52")
	ip2 := s.Find(7)
	fmt.Println("IP is ", ip2)

	s.Insert(1, "192.168.0.1")
	s.Insert(2, "192.168.31.52")
	s.Insert(3, "192.168.31.52")
	s.Insert(4, "192.168.31.54")

	query := []SectorId{1, 2, 3, 4, 5, 6}
	ans := s.FindAllSort(query)
	fmt.Println("IP is ", ans)
}

func TestB(t *testing.T) {

	s := GetIns()
	s.Insert(5, "192.168.0.1")
	ip := s.Find(5)
	fmt.Println("IP is ", ip)

	s.Insert(6, "192.168.31.52")
	ip2 := s.Find(7)
	fmt.Println("IP is ", ip2)

	s.Insert(1, "192.168.0.1")
	s.Insert(2, "192.168.31.52")
	s.Insert(3, "192.168.31.52")
	s.Insert(4, "192.168.31.54")

	query := []SectorId{1, 2, 3, 4, 5, 6}
	ans := s.FindAllSort(query)
	fmt.Println("IP is ", ans)
}