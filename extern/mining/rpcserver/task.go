package umrpc

type Sector struct {

}

type SectorTaskRequest struct {
	TaskId int64
	TaskName string
	Params []string
}

type SectorTaskRespose struct {
	TaskId int64
	TaskName string
	ReturnValue []string
}

func (this *Sector) SectorSeed(req SectorTaskRequest, res *SectorTaskRespose) error {

	return nil
}