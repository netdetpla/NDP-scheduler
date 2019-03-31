package main

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"time"
)

var scanOpt = make(chan string, 50)

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

func checkImage(db *sql.DB, taskID int) (err error) {
	checkImageSQL := "select is_loaded from image where image.id in (select image_id from task where id = ?)"
	rows, err := db.Query(checkImageSQL, taskID)
	if err != nil {
		return
	}
	if !rows.Next() {
		return errors.New("image was not found")
	}
	var isLoaded int
	err = rows.Scan(&isLoaded)
	if err != nil {
		return
	}
	if isLoaded == 1 {
		return
	} else {
		uploadOneImageSQL := "select image_name, tag, file_name from image where id in (select image_id from task where id = ?)"
		rows, err := db.Query(uploadOneImageSQL, taskID)
		if err != nil {
			return
		}
		var uploadOneImage []*imageInfo
		for !rows.Next() {
			i := new(imageInfo)
			err = rows.Scan(&i.imageName, &i.tag, &i.fileName)
			if err != nil {
				return
			}
			uploadOneImage = append(uploadOneImage, i)
		}
		// TODO docker操作

		return
	}
}

func scanTask(db *sql.DB) (err error) {
	taskSQL := "select task.id, image.image_name, image.tag, task.param from task, image where task.task_status = 20000 and task.image_id = image.id"
	rows, err := db.Query(taskSQL)
	if err != nil {
		return
	}
	var tasks []*taskInfo
	for !rows.Next() {
		i := new(taskInfo)
		err = rows.Scan(&i.id, &i.imageName, &i.tag, &i.param)
		if err != nil {
			return
		}
		tasks = append(tasks, i)
	}
	// TODO 下发操作

	return
}

// 查询未
func scanImage(db *sql.DB) (err error) {
	imageSQL := "select image_name, tag, file_name from image where is_loaded = 0"
	rows, err := db.Query(imageSQL)
	if err != nil {
		return
	}
	var waitLoadingImages []*imageInfo
	for !rows.Next() {
		i := new(imageInfo)
		err = rows.Scan(&i.imageName, &i.tag, &i.fileName)
		if err != nil {
			return
		}
		waitLoadingImages = append(waitLoadingImages, i)
	}
	// TODO docker操作

	return
}

// 任务扫描定时器
func taskTimer() {
	for true {
		time.Sleep(30 * time.Second)
		scanOpt <- "task"
	}
}

// 镜像扫描定时器
func imageTimer() {
	for true {
		time.Sleep(600 * time.Second)
		scanOpt <- "image"
	}
}

// 根据定时器信号启动对应的数据库查询操作
func databaseScanner(databaseInfo *database) (err error) {
	databaseURL := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?timeout=20s",
		databaseInfo.Username,
		databaseInfo.Password,
		databaseInfo.Host,
		databaseInfo.Port,
		databaseInfo.DatabaseName)
	db, err := sql.Open("mysql", databaseURL)
	if err != nil {
		return
	}
	// 测试数据库连接
	if err = db.Ping(); err != nil {
		return
	}
	// 启动定时器
	go taskTimer()
	go imageTimer()
	for true {
		// 读取channel里消息并调用对应方法，没有则阻塞等待
		switch <-scanOpt {
		case "image":
			err = scanImage(db)
			if err != nil {
				return
			}
		case "task":
			err = scanTask(db)
			if err != nil {
				return
			}
		}
	}
	return
}
