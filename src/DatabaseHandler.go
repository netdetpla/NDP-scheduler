package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"time"
)

type dbOpt struct {
	operation string
	param     []string
}

type imageInfo struct {
	imageName string
	tag       string
	fileName  string
}

type taskInfo struct {
	id        int
	imageName string
	tag       string
	param     string
}

var scanOpt = make(chan dbOpt, 50)
var checkReq = make(chan int, 1)

func checkImage(db *sql.DB, taskID int) (checkFlag bool) {
	checkImageSQL := "select is_loaded from image where image.id in (select image_id from task where id = ?)"
	rows, err := db.Query(checkImageSQL, taskID)
	if err != nil {
		return
	}
	if !rows.Next() {
		return false
	}
	var isLoaded int
	err = rows.Scan(&isLoaded)
	if err != nil {
		return false
	}
	if isLoaded == 1 {
		return true
	} else {
		uploadOneImageSQL := "select image_name, tag, file_name from image where id in (select image_id from task where id = ?)"
		rows, err := db.Query(uploadOneImageSQL, taskID)
		if err != nil {
			return false
		}
		for !rows.Next() {
			i := new(imageInfo)
			err = rows.Scan(&i.imageName, &i.tag, &i.fileName)
			if err != nil {
				return false
			}
			loadOpt <- *i
		}
		return false
	}
}

func updateImageLoadedStatus(db *sql.DB, imageName string, tag string) (err error) {
	updateSQL := "update image set is_loaded = 1 where image_name = ? and tag = ?"
	_, err = db.Exec(updateSQL, imageName, tag)
	return
}

func scanTask(db *sql.DB) (err error) {
	taskSQL := "select task.id, image.image_name, image.tag, task.param from task, image where task.task_status = 20000 and task.image_id = image.id"
	rows, err := db.Query(taskSQL)
	if err != nil {
		return
	}
	for !rows.Next() {
		i := new(taskInfo)
		err = rows.Scan(&i.id, &i.imageName, &i.tag, &i.param)
		if err != nil {
			return
		}
		if checkImage(db, i.id) {
			taskQueue <- *i
		}
	}
	return
}

// 查询未
func scanImage(db *sql.DB) (err error) {
	imageSQL := "select image_name, tag, file_name from image where is_loaded = 0"
	rows, err := db.Query(imageSQL)
	if err != nil {
		return
	}
	for !rows.Next() {
		i := new(imageInfo)
		err = rows.Scan(&i.imageName, &i.tag, &i.fileName)
		if err != nil {
			return
		}
		loadOpt <- *i
	}
	return
}

// 任务扫描定时器
func taskTimer() {
	for true {
		time.Sleep(30 * time.Second)
		scanOpt <- dbOpt{"task", []string{}}
	}
}

// 镜像扫描定时器
func imageTimer() {
	for true {
		time.Sleep(600 * time.Second)
		scanOpt <- dbOpt{"image", []string{}}
	}
}

// 根据定时器信号启动对应的数据库查询操作
func databaseScanner(databaseInfo *database) {
	fmt.Println("dbh start")
	databaseURL := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?timeout=20s",
		databaseInfo.Username,
		databaseInfo.Password,
		databaseInfo.Host,
		databaseInfo.Port,
		databaseInfo.DatabaseName)
	db, err := sql.Open("mysql", databaseURL)
	if err != nil {
		// TODO 错误处理
		return
	}
	// 测试数据库连接
	if err = db.Ping(); err != nil {
		// TODO 错误处理
		return
	}
	// 启动定时器
	go taskTimer()
	go imageTimer()
	for {
		// 读取channel里消息并调用对应方法，没有则阻塞等待
		so := <-scanOpt
		switch so.operation {
		case "image":
			err = scanImage(db)
			if err != nil {
				// TODO 错误处理
				continue
			}
		case "task":
			err = scanTask(db)
			if err != nil {
				// TODO 错误处理
				continue
			}
		case "loaded":
			err = updateImageLoadedStatus(db, so.param[0], so.param[1])
			if err != nil {
				// TODO 错误处理
				continue
			}
		}
	}
}
