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
	priority  int
}

var scanOpt = make(chan dbOpt, 50)

func updateTaskStatus(db *sql.DB, status, taskID string) (err error) {
	updateTaskStatusSQL := "update task set task_status = ? where id = ?"
	_, err = db.Exec(updateTaskStatusSQL, status, taskID)
	return
}

func checkImage(db *sql.DB, taskID int) (checkFlag bool) {
	checkImageSQL := "select is_loaded from image where image.id in (select image_id from task where id = ?)"
	var isLoaded int
	err := db.QueryRow(checkImageSQL, taskID).Scan(&isLoaded)
	if err != nil {
		log.Warning(err.Error())
		return false
	}
	if isLoaded == 1 {
		return true
	} else {
		uploadOneImageSQL := "select image_name, tag, file_name from image where id in (select image_id from task where id = ?)"
		rows, err := db.Query(uploadOneImageSQL, taskID)
		if err != nil {
			log.Warning(err.Error())
			return false
		}
		for rows.Next() {
			i := new(imageInfo)
			err = rows.Scan(&i.imageName, &i.tag, &i.fileName)
			if err != nil {
				return false
			}
			loadOpt <- *i
		}
		err = rows.Close()
		if err != nil {
			log.Warning(err.Error())
			return false
		}
		return false
	}
}

func updateImageLoadedStatus(db *sql.DB, imageName string, tag string) (err error) {
	updateSQL := "update image set is_loaded = 1 where image_name = ? and tag = ?"
	_, err = db.Exec(updateSQL, imageName, tag)
	return
}

func scanTaskStatus(db *sql.DB) (err error) {
	runningSQL := "select count(id) from task where task_status = 20020 limit 1"
	var result int
	err = db.QueryRow(runningSQL).Scan(&result)
	if err != nil || result == 1{
		return
	}
	scanOpt <- dbOpt{"task", []string{}}
	return
}
func scanTask(db *sql.DB) (err error) {
	minPrioritySQL := "select MIN(priority) from task where task_status = 20000"
	var minPriority sql.NullInt64
	err = db.QueryRow(minPrioritySQL).Scan(&minPriority)
	if err != nil {
		log.Warning(err.Error())
		return
	}
	if !minPriority.Valid {
		return
	}
	taskInfoSQL := "select id, image_id, param from task where priority = ? and task_status = 20000 limit 1"
	i := new(taskInfo)
	i.priority = int(minPriority.Int64)
	var imageID int
	err = db.QueryRow(taskInfoSQL, minPriority).Scan(&i.id, &imageID, &i.param)
	if err != nil {
		log.Warning(err.Error())
		return
	}
	imageInfoSQL := "select image_name, tag from image where id = ?"
	err = db.QueryRow(imageInfoSQL, imageID).Scan(&i.imageName, &i.tag)
	if err != nil {
		log.Warning(err.Error())
		return
	}
	if checkImage(db, i.id) {
		taskQueue <- *i
	}
	return
}

// 查询未
func scanImage(db *sql.DB) (err error) {
	imageSQL := "select image_name, tag, file_name from image where is_loaded = 0"
	rows, err := db.Query(imageSQL)
	if err != nil {
		fmt.Print(err.Error())
		return
	}
	defer func() {
		err = rows.Close()
	}()
	for rows.Next() {
		i := new(imageInfo)
		err = rows.Scan(&i.imageName, &i.tag, &i.fileName)
		if err != nil {
			fmt.Print(err.Error())
			return
		}
		loadOpt <- *i
	}
	return
}

func insertResult(db *sql.DB, resultLine string, taskID string, table string) (err error) {
	log.Debug("new result", table, taskID, resultLine)
	resultSQL := "insert into " + table + " (task_id, result_line) values (?, ?)"
	_, err = db.Exec(resultSQL, taskID, resultLine)
	return
}

// 任务扫描定时器
func taskTimer() {
	for true {
		time.Sleep(10 * time.Second)
		scanOpt <- dbOpt{"status", []string{}}
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
	log.Notice("database scanner started.")
	// 连接建立
	databaseURL := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?timeout=20s",
		databaseInfo.Username,
		databaseInfo.Password,
		databaseInfo.Host,
		databaseInfo.Port,
		databaseInfo.DatabaseName)
	mysqlDB, err := sql.Open("mysql", databaseURL)
	if err != nil {
		log.Warning(err.Error())
		return
	}
	defer func() {
		err = mysqlDB.Close()
		if err != nil {
			log.Warning(err.Error())
		}
	}()
	// 测试数据库连接
	if err = mysqlDB.Ping(); err != nil {
		log.Warning(err.Error())
		return
	}
	// 启动定时器
	go taskTimer()
	go imageTimer()
	scanOpt <- dbOpt{"image", []string{}}
	scanOpt <- dbOpt{"status", []string{}}
	for {
		// 读取channel里消息并调用对应方法，没有则阻塞等待
		so := <-scanOpt
		switch so.operation {
		case "image":
			err = scanImage(mysqlDB)
		case "status":
			err = scanTaskStatus(mysqlDB)
		case "task":
			err = scanTask(mysqlDB)
		case "loaded":
			err = updateImageLoadedStatus(mysqlDB, so.param[0], so.param[1])
		case "task-status":
			err = updateTaskStatus(mysqlDB, so.param[0], so.param[1])
		case "result":
			err = insertResult(mysqlDB, so.param[0], so.param[1], so.param[2])
		default:
			log.Debug(so.operation, so.param)
			if err != nil {
				log.Warning("operation error: " + err.Error())
			}
		}
	}
}
