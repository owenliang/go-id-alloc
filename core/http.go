package core

import (
	"time"
	"net"
	"net/http"
	"strconv"
	"fmt"
)

func handleAlloc(w http.ResponseWriter, r *http.Request) {
	nextId, err := GMysql.NextId()
	if err == nil{
		strNextId := fmt.Sprintf("%d", nextId)
		w.Write([]byte(strNextId))
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