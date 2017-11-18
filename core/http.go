package core

import (
	"time"
	"net"
	"net/http"
	"strconv"
	"fmt"
	"encoding/json"
)

type allocResponse struct {
	Errno int
	Msg string
	Id int64
}

func handleAlloc(w http.ResponseWriter, r *http.Request) {
	resp := allocResponse{}

	if nextId, err := GAlloc.NextId(); err != nil {
		resp.Errno = -1
		resp.Msg = fmt.Sprintf("%v", err)
	} else {
		resp.Id = nextId
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