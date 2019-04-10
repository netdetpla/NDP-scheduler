package main

import (
	"container/heap"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"time"
)

type sendTaskJSON struct {
	ID        string `json:"id"`
	ImageName string `json:"image-name"`
	Tag       string `json:"tag"`
	Param     string `json:"param"`
}

var taskQueue = make(chan taskInfo, 50)

func taskSender(addr *server, t taskInfo) (err error) {
	d := sendTaskJSON{strconv.Itoa(t.id), t.imageName, t.tag, t.param}
	b, err := json.Marshal(d)
	//TODO 节点选择，目前不实现
	conn, err := net.Dial("tcp", addr.Ip+":"+addr.Port)
	if err != nil {
		// TODO 错误处理
		fmt.Print(err.Error())
		return
	}
	_, err = conn.Write(b)
	if err != nil {
		// TODO 错误处理
		fmt.Print(err.Error())
	}
	return
}

const MaxLen = 20

type taskItem struct {
	task     *taskInfo
	priority int
	index    int
}

type taskPriorityQueue []*taskItem

func (tq taskPriorityQueue) Len() int {
	return len(tq)
}

func (tq taskPriorityQueue) Less(i, j int) bool {
	return tq[i].priority > tq[j].priority
}

func (tq taskPriorityQueue) Swap(i, j int) {
	tq[i], tq[j] = tq[j], tq[i]
	tq[i].index, tq[j].index = i, j
}

func (tq *taskPriorityQueue) Pop() interface{} {
	n := len(*tq)
	t := (*tq)[n-1]
	t.index = -1
	*tq = (*tq)[0 : n-1]
	return t
}

func (tq *taskPriorityQueue) Push(x interface{}) {
	n := len(*tq)
	t := x.(*taskItem)
	t.index = n
	*tq = append(*tq, t)
}

func handleNewTask(tq *taskPriorityQueue, t *taskInfo) {
	n := len(*tq)
	if n > MaxLen {
		// 队列已满
		if t.priority < (*tq)[n-1].priority {
			// 队外任务优先级高
			(*tq)[n-1].priority = t.priority
			(*tq)[n-1].task = t
			scanOpt <- dbOpt{"task-status", []string{
				strconv.Itoa(t.id),
				strconv.Itoa(20010)}}
			scanOpt <- dbOpt{"task-status", []string{
				strconv.Itoa((*tq)[n-1].task.id),
				strconv.Itoa(20000)}}
		}
	} else {
		// 队列未满
		heap.Push(tq, t)
		scanOpt <- dbOpt{"task-status", []string{
			strconv.Itoa(t.id),
			strconv.Itoa(20010)}}
	}
}

func taskQueueManager(addr *server) {
	log.Notice("task queue manager started.")
	tq := make(taskPriorityQueue, 0)
	//tq[0] = &taskItem{
	//	task: &taskInfo{
	//		imageName: "",
	//		tag:       "",
	//		id:        0,
	//		param:     ""},
	//	priority: 11,
	//	index:    0}
	//log.Debug("tq: ", tq)
	heap.Init(&tq)
	//heap.Pop(&tq)

	/*
		大致思路为消费taskQueue中任务加入优先级队列，然后开始下发
		1. 入队阶段
		1.1 队列未满，t直接入队
		1.2 队列已满，判断t的优先级是否高于队列中最低优先级，是则替换，否则丢弃
		1.3 等待超时则进入下发阶段
		2 下发阶段
		2.1* 判断资源是否满足最高优先级任务，否则回到入队阶段
		2.2 下发最高优先级任务到节点，到2.1
	*/
	for {
		// 入队阶段
		timer := time.NewTimer(10 * time.Second)
		select {
		case t := <-taskQueue:
			handleNewTask(&tq, &t)
		case <-timer.C:
		default:
			timer.Stop()
		}
		// 下发阶段
		// TODO 资源判断
		for len(tq) > 0 {
			err := taskSender(addr, *heap.Pop(&tq).(*taskItem).task)
			if err != nil {
				// TODO 错误处理
				log.Warning(err.Error())
				break
			}
		}
	}
}
