package main

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type server struct {
	ip string
	port int
}

type database struct {
	host string
	port int
	databaseName string
	username string
	password string
}

type config struct {
	title    string
	Database database `toml:"Database"`
	Server server `toml:"server"`
}

// 获取当前绝对路径
func GetAppPath() string {
    file, _ := exec.LookPath(os.Args[0])
    path, _ := filepath.Abs(file)
    index := strings.LastIndex(path, string(os.PathSeparator))
    return path[:index + 1]
}

// 读取config文件
func GetConfig() (configInfo config) {
	b, err := ioutil.ReadFile(GetAppPath() + "config.toml")
    if err != nil {
        fmt.Print(err.Error())
    }
    fmt.Println(b)
    configString := string(b)
	_, err = toml.Decode(configString, &configInfo)
	if err != nil {
		fmt.Print(err.Error())
	}
	return
}
