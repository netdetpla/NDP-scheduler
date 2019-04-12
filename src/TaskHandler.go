package main

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
)

type sendTaskJSON struct {
	ID        string `json:"id"`
	ImageName string `json:"image_name"`
	Tag       string `json:"tag"`
	Param     string `json:"param"`
	Config map[string]string `json:"config"`
	TaskTag string `json:"task_tag"`
}

var taskQueue = make(chan taskInfo, 50)

func taskSender(addr *server, t taskInfo) (err error) {
	d := sendTaskJSON{strconv.Itoa(t.id), t.imageName, t.tag, t.param, map[string]string{}, "task"}
	b, err := json.Marshal(d)
	//TODO 节点选择，目前不实现
	conn, err := net.Dial("tcp", addr.Ip+":"+addr.Port)
	if err != nil {
		// TODO 错误处理
		fmt.Print(err.Error())
		return
	}
	defer func() {
		err = conn.Close()
		if err != nil {
			log.Warning(err.Error())
		}
	}()

	_, err = conn.Write(b)
	if err != nil {
		// TODO 错误处理
		fmt.Print(err.Error())
		return
	}
	scanOpt <- dbOpt{"task-status", []string{strconv.Itoa(20010), strconv.Itoa(t.id)}}
	return
}

func taskQueueManager(addr *server) {
	log.Notice("task queue manager started.")
	for {
		t := <-taskQueue
		// TODO 资源判断
		err := taskSender(addr, t)
		if err != nil {
			// TODO 错误处理
			log.Warning(err.Error())
		}
	}
}
