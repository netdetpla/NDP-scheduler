package main

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

type param struct {
	Parameter string `json:"parameter"`
}

type dispatch struct {
	ImageName string  `json:"image_name"`
	ParamList []param `json:"parameter_list"`
	BusiName string `json:"busi_name"`
	Hash string `json:"hash"`
}

var taskQueue = make(chan taskInfo, 50)

func taskSender(addr *server) {
	fmt.Println("th start")
	for {
		conn, err := net.Dial("tcp", addr.Ip + ":" + addr.Port)
		if err != nil {
			// TODO 错误处理
			fmt.Print(err.Error())
			time.Sleep(time.Second * 10)
			continue
		}
		t := <-taskQueue
		d := dispatch{t.imageName + ":" + t.tag, []param{{t.param}}, "iie", "abcd"}
		b, err := json.Marshal(d)
		if err != nil {
			// TODO 错误处理
			fmt.Print(err.Error())
			time.Sleep(time.Second * 10)
			continue
		}
		_, err = conn.Write(b)
		// TODO 错误处理
		time.Sleep(time.Second * 10)
	}
}
