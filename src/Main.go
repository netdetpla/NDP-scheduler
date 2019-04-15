package main

import (
	"github.com/op/go-logging"
	"os"
)
// 日志
var log = logging.MustGetLogger("example")
var format = logging.MustStringFormatter(
	`%{color}%{time:15:04:05} %{shortfile} %{shortfunc} ▶ %{level:.4s} %{color:reset}  %{message}`,
)

func init() {
	// 日志初始化配置
	backend1 := logging.NewLogBackend(os.Stderr, "", 0)
	backend2 := logging.NewLogBackend(os.Stderr, "", 0)
	backend2Formatter := logging.NewBackendFormatter(backend2, format)
	backend1Leveled := logging.AddModuleLevel(backend1)
	backend1Leveled.SetLevel(logging.ERROR, "")
	logging.SetBackend(backend1Leveled, backend2Formatter)
}
func main() {
	config := GetConfig()
	go databaseScanner(&config.Database)
	go imageLoader(&config.Image)
	go taskQueueManager(&config.Server)
	go consumersManager()
	select {}
}
