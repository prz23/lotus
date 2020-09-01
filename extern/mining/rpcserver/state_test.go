package umrpc

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

//testB inertface jicheng
type interfaceA interface {
	function01()
}

type A struct {
	ff   interfaceA
	num int64
}

type B struct {
	num int64
}

func (i *B) function01() {
	fmt.Println("B -> interface A")
}

type C struct {
	num int64
}

func (i *C) function01() {
	fmt.Println("C -> interface A")
}


type D struct {
	A
	num int64
}

func (i *D) function01() {
	fmt.Println("D -> interface A")
}

func TestB(t *testing.T) {
	a := A{
		ff:   &B{num: 4},
		num: 5,
	}
	a.ff.function01()
}

func TestC(t *testing.T) {
	a := A{
		ff:   &B{num: 4},
		num: 5,
	}
	d := D{
		a,
		5,
	}
	d.ff.function01()
	d.function01()
}

type interfaceB interface {
	interfaceC
	function02()
}

type E struct {
	a interfaceB
	num int64
}
type F struct {
}

func (s *F)function02(){
	fmt.Println("F -> intB")
}

type interfaceC interface {
	function03()
}

func (s *F)function03(){
	fmt.Println("F -> intC")
}

func TestD(t *testing.T) {
	f := F{}
    e := E{
		a:   &f,
		num: 0,
	}
	e.a.function03()
}

func TestE(t *testing.T) {
	StartMasterRpc2("127.0.0.1:8888")
}