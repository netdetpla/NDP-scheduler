package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// scanservice
type scanServicePort struct {
	Version  string `json:"version"`
	Port     string `json:"port"`
	Protocol string `json:"protocol"`
	Service  string `json:"service"`
	Product  string `json:"product"`
}
type scanServiceResult struct {
	Ports     []scanServicePort `json:"ports"`
	OSVersion string            `json:"os_version"`
	IP        string            `json:"ip"`
	Hardware  string            `json:"hardware"`
}
type scanService struct {
	Result    []scanServiceResult `json:"result"`
	SubtaskID string              `json:"subtask_id"`
	TaskID    string              `json:"task_id"`
	TaskName  string              `json:"task_name"`
}

func ParseScanService(db *sql.DB, resultLine string) (err error) {
	result := new(scanService)
	err = json.Unmarshal([]byte(resultLine), result)
	if err != nil {
		fmt.Print(err.Error())
		return
	}
	for _, r := range result.Result {
		// 直接更新结果
		replaceIPSQL := "replace into `ip`(`ip`, `os_version`, `harward`) values (?, ?, ?)"
		_, err = db.Exec(replaceIPSQL, r.IP, r.OSVersion, r.Hardware)
		if err != nil {
			fmt.Print(err.Error())
			return
		}
		// 获取ip对应id
		ipIDSQL := "select id from `ip` where `ip` = ?"
		var id int
		err = db.QueryRow(ipIDSQL).Scan(&id)
		if err != nil {
			fmt.Print(err.Error())
			return
		}
		for _, p := range r.Ports {
			// 更新port表
			replacePortSQL := "replace into `port`(`ip_id`, `port`, `protocol`, `service`, `product`, `version`) " +
				"values (?, ?, ?, ?, ?, ?)"
			_, err = db.Exec(replacePortSQL, id, p.Port, p.Protocol, p.Service, p.Product, p.Version)
			if err != nil {
				fmt.Print(err.Error())
				return
			}
		}
	}
	return
}
