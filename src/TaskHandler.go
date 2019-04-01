package main

import (
	"encoding/json"
	"net"
	"strconv"
	"time"
)

type param struct {
	Parameter string `json:"parameter"`
}

type dispatch struct {
	ImageName string  `json:"image_name"`
	ParamList []param `json:"parameter_list"`
}

var taskQueue = make(chan taskInfo, 50)

func taskSender(addr *server) {
	for {
		portStr := strconv.FormatFloat(addr.Port, 'f', -1, 64)
		conn, err := net.Dial("tcp", addr.Ip+":"+portStr)
		if err != nil {
			// TODO 错误处理
			time.Sleep(time.Second * 10)
			continue
		}
		t := <-taskQueue
		d := dispatch{t.imageName, []param{{t.param}}}
		b, err := json.Marshal(d)
		if err != nil {
			// TODO 错误处理
			time.Sleep(time.Second * 10)
			continue
		}
		_, err = conn.Write(b)
		// TODO 错误处理
		time.Sleep(time.Second * 10)
	}
}
