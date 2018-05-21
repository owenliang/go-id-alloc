package core

import (
	"io/ioutil"
	"encoding/json"
)

type Conf struct {
	DSN string	`json:"DSN"`
	Table string `json:"table"`

	HttpPort int `json:"httpPort"`
	HttpReadTimeout int `json:"httpReadTimeout"`
	HttpWriteTimeout int	`json:"httpWriteTimeout"`
}

var GConf *Conf

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