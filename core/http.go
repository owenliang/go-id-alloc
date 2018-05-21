package core

import (
	"time"
	"net"
	"net/http"
	"strconv"
	"encoding/json"
	"fmt"
	"errors"
)

type allocResponse struct {
	Errno int `json:"errno"`
	Msg string	`json:"msg"`
	Id int64	`json:"id"`
}

type healthResponse struct {
	Errno int `json:"errno"`
	Msg string	`json:"msg"`
	Left int64 `json:"left"`
}

func handleAlloc(w http.ResponseWriter, r *http.Request) {
	var (
		resp allocResponse = allocResponse{}
		err error
		bytes []byte
		bizTag string
	)

	if err = r.ParseForm(); err != nil {
		goto RESP
	}

	if bizTag = r.Form.Get("biz_tag"); bizTag == "" {
		err = errors.New("need biz_tag param")
		goto RESP
	}

	for { // 跳过ID=0, 一般业务不支持ID=0
		if resp.Id, err = GAlloc.NextId(bizTag); err != nil {
			goto RESP
		}
		if resp.Id != 0 {
			break
		}
	}

RESP:
	if err != nil {
		resp.Errno = -1
		resp.Msg = fmt.Sprintf("%v", err)
		w.WriteHeader(500)
	} else {
		resp.Msg = "success"
	}
	if bytes, err = json.Marshal(&resp); err == nil {
		w.Write(bytes)
	} else {
		w.WriteHeader(500)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	var (
		resp healthResponse = healthResponse{}
		err error
		bizTag string
	)

	if err = r.ParseForm(); err != nil {
		goto RESP
	}

	if bizTag = r.Form.Get("biz_tag"); bizTag == "" {
		err = errors.New("need biz_tag param")
		goto RESP
	}

	resp.Left = GAlloc.LeftCount(bizTag)
	if resp.Left == 0 {
		err = errors.New("no available id ")
		goto RESP
	}

RESP:
	if err != nil {
		resp.Errno = -1
		resp.Msg = fmt.Sprintf("%v", err)
		w.WriteHeader(500)
	} else {
		resp.Msg = "success"
	}
	if bytes, err := json.Marshal(&resp); err == nil {
		w.Write(bytes)
	} else {
		w.WriteHeader(500)
	}
}

func StartServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/alloc", handleAlloc)
	mux.HandleFunc("/health", handleHealth)

	srv := &http.Server{
		ReadTimeout: time.Duration(GConf.HttpReadTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(GConf.HttpWriteTimeout) * time.Millisecond,
		Handler: mux,
	}
	listener, err := net.Listen("tcp", ":" + strconv.Itoa(GConf.HttpPort))
	if err != nil {
		return err
	}
	return srv.Serve(listener)
}