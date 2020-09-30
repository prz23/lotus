package rpctypes

type LocalServerAddr string
type RemoteServerAddr string

type Role int

const (
	Role_Master Role = 1
	Role_Slave  Role = 2
)