package main

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
)

type sendTaskJSON struct {
	ID        string            `json:"id"`
	ImageName string            `json:"image_name"`
	Tag       string            `json:"tag"`
	Param     string            `json:"param"`
	Config    map[string]string `json:"config"`
	TaskTag   string            `json:"task_tag"`
}

var taskQueue = make(chan taskInfo, 50)

func taskSender(addr *server, t taskInfo) (err error) {
	unlockFlag := true
	// 退出前如果任务下发失败则解锁
	defer func() {
		if unlockFlag {
			_ = unlockExecutor(mysqlDB, t.executorIP)
			return
		}
	}()
	d := sendTaskJSON{strconv.Itoa(t.id), t.imageName, t.tag, t.param, map[string]string{}, "task"}
	b, err := json.Marshal(d)
	//conn, err := net.Dial("tcp", addr.Ip+":"+addr.Port)
	conn, err := net.Dial("tcp", t.executorIP+":"+addr.Port)
	if err != nil {
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
	unlockFlag = false
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
