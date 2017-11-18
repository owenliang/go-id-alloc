package core

import (
	"io/ioutil"
	"encoding/json"
)

type Conf struct {
	PartitionIdx int
	TotalPartition int
	SegmentSize int
	DSN string

	HttpPort int
	HttpReadTimeout int
	HttpWriteTimeout int
}

var GConf *Conf = nil

func LoadConf(filename string) error {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	conf := Conf{}
	err = json.Unmarshal(content, &conf)
	if err != nil {
		return err
	}
	GConf = &conf
	return nil
}