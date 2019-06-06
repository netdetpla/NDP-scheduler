package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type server struct {
	Ip   string  `json:"ip"`
	Port string `json:"port"`
}

type database struct {
	Host         string  `json:"host"`
	Port         string `json:"port"`
	DatabaseName string  `json:"database_name"`
	Username     string  `json:"username"`
	Password     string  `json:"password"`
}

type image struct {
	Path string `json:"path"`
	RepoPort string `json:"repo_port"`
}

type resource struct {
	Port string `json:"port"`
}

type config struct {
	Server   server   `json:"server"`
	Image    image    `json:"image"`
	Database database `json:"database"`
	Resource resource `json:"resource"`
}

// 获取当前绝对路径
func GetAppPath() string {
	file, _ := exec.LookPath(os.Args[0])
	path, _ := filepath.Abs(file)
	index := strings.LastIndex(path, string(os.PathSeparator))
	return path[:index+1]
}

// 读取config文件
func GetConfig() (configInfo *config) {
	configInfo = new(config)
	b, err := ioutil.ReadFile(GetAppPath() + "config.json")
	if err != nil {
		fmt.Print(err.Error())
	}
	err = json.Unmarshal(b, configInfo)
	if err != nil {
		fmt.Print(err.Error())
	}
	return
}
