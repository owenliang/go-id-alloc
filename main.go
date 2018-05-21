package main

import (
	"runtime"
	"flag"
	"fmt"
	"os"
	"github.com/owenliang/go-id-alloc/core"
)

var (
	configFile string
)

func initCmd() {
	flag.StringVar(&configFile, "config", "./alloc.json", "where alloc.json is.")
	flag.Parse()
}

func initEnv() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	initEnv()
	initCmd()

	var err error = nil

	if err = core.LoadConf(configFile); err != nil {
		goto ERROR;
	}
	if err = core.InitMysql(); err != nil {
		goto ERROR;
	}

	if err = core.InitAlloc(); err != nil {
		goto ERROR;
	}

	if err = core.StartServer(); err != nil {
		goto ERROR;
	}

	os.Exit(0)
ERROR:
	fmt.Println(err)
	os.Exit(-1)
}
