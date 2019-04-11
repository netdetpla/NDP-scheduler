package main

import (
	"database/sql"
	"fmt"
	"github.com/amsokol/ignite-go-client/binary/v1"
	_ "github.com/amsokol/ignite-go-client/sql"
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
var igniteCache = "ndpTaskCache"

func initIgniteTable(c *ignite.Client) {
	query := "CREATE TABLE task (" +
		"id int PRIMARY KEY," +
		"task_name char(128)," +
		"image_id int," +
		"start_time char(32)," +
		"end_time char(32)," +
		"param text," +
		"task_status int," +
		"priority int," +
		"tid int)"
	_, err := (*c).QuerySQL(igniteCache, false, ignite.QuerySQLData{Query: query})
	if err != nil {
		log.Warning(err.Error())
		return
	}
}

func updateTaskStatus(db *sql.DB, status, taskID string) (err error) {
	updateTaskStatusSQL := "update task set task_status = ? where id = ?"
	_, err = db.Exec(updateTaskStatusSQL, status, taskID)
	return
}

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
		for rows.Next() {
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

func scanTask(c *ignite.Client, db *sql.DB) (err error) {
	taskSQL := "select id, task_name, image_id, param, priority from task where id in " +
		"(select MIN(priority) from task limit 1)"
	taskFromIgnite, err := (*c).QuerySQL(igniteCache, false, ignite.QuerySQLData{Query: taskSQL})
	if err != nil {
		log.Warning(err.Error())
		return
	}
	log.Debug(taskFromIgnite.Rows)
	//if checkImage(db, i.id) {
	//	taskQueue <- *i
	//}
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

// 任务扫描定时器
func taskTimer() {
	for true {
		time.Sleep(60 * time.Second)
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
	igniteDB, err := ignite.Connect(ignite.ConnInfo{
		Network: "tcp",
		Host:    "127.0.0.1",
		Port:    10800,
		Major:   1,
		Minor:   1,
		Patch:   0})
	if err != nil {
		log.Warning(err.Error())
		return
	}
	defer func() {
		err = mysqlDB.Close()
		if err != nil {
			log.Warning(err.Error())
		}
		err = igniteDB.Close()
		if err != nil {
			log.Warning(err.Error())
		}
	}()
	// 测试数据库连接
	if err = mysqlDB.Ping(); err != nil {
		log.Warning(err.Error())
		return
	}
	// 创建ignite cache
	err = igniteDB.CacheCreateWithName(igniteCache)
	if err != nil {
		log.Warning(err.Error())
		return
	}
	// 初始化ignite task表
	initIgniteTable(&igniteDB)
	// 启动定时器
	go taskTimer()
	go imageTimer()
	scanOpt <- dbOpt{"image", []string{}}
	scanOpt <- dbOpt{"task", []string{}}
	for {
		// 读取channel里消息并调用对应方法，没有则阻塞等待
		so := <-scanOpt
		switch so.operation {
		case "image":
			err = scanImage(mysqlDB)
		case "task":
			err = scanTask(&igniteDB, mysqlDB)
		case "loaded":
			err = updateImageLoadedStatus(mysqlDB, so.param[0], so.param[1])
		case "task-status":
			err = updateTaskStatus(mysqlDB, so.param[0], so.param[1])
		default:
			log.Debug(so.operation, so.param)
			if err != nil {
				log.Warning("operation error: " + err.Error())
			}
		}
	}
}
