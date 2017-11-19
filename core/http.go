package core

import (
	"time"
	"net"
	"net/http"
	"strconv"
	"encoding/json"
)

type allocResponse struct {
	Errno int
	Msg string
	Id int64
}

func handleAlloc(w http.ResponseWriter, r *http.Request) {
	var (
		resp allocResponse = allocResponse{}
		err error
		bytes []byte
	)

	for { // skip Id=0
		if resp.Id, err = GAlloc.NextId(); err != nil {
			w.WriteHeader(500)
			return
		}
		if resp.Id != 0 {
			break
		}
	}

	if bytes, err = json.Marshal(&resp); err == nil {
		w.Write(bytes)
	} else {
		w.WriteHeader(500)
	}
}

func StartServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/alloc", handleAlloc)

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