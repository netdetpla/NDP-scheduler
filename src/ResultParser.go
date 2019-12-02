package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"strings"
)

// scanservice
type scanServicePort struct {
	Version  string `json:"version"`
	Port     int64  `json:"port"`
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

func InetAtoN(ip string) int64 {
	ret := big.NewInt(0)
	ret.SetBytes(net.ParseIP(ip).To4())
	return ret.Int64()
}
func ParseScanService(db *sql.DB, resultLine string) (err error) {
	result := new(scanService)
	err = json.Unmarshal([]byte(resultLine), result)
	if err != nil {
		fmt.Print(err.Error())
		return
	}
	for _, r := range result.Result {
		intIP := InetAtoN(r.IP)
		// 查找ip地理坐标
		selectGeoSQL := "select geoname_id from GeoLite2-City-Blocks-IPv4 where long_ip_start <= ? and long_ip_end >= ?"
		var geoID sql.NullInt64
		err = db.QueryRow(selectGeoSQL).Scan(&geoID)
		if err != nil {
			log.Warning(err.Error())
			return
		}
		if !geoID.Valid {
			return
		}
		// 直接更新结果
		replaceIPSQL := "replace into `ip`(`id`, `ip`, `os_version`, `hardware`, `lnglat_id`) values (?, ?, ?, ?, ?)"
		_, err = db.Exec(replaceIPSQL, intIP, r.IP, r.OSVersion, r.Hardware, geoID)
		if err != nil {
			log.Error(err.Error())
			return
		}
		for _, p := range r.Ports {
			// 更新port表
			replacePortSQL := "replace into `port`(`ip_id`, `port`, `protocol`, `service`, `product`, `version`) " +
				"values (?, ?, ?, ?, ?, ?)"
			_, err = db.Exec(replacePortSQL, intIP, p.Port, p.Protocol, p.Service, p.Product, p.Version)
			if err != nil {
				log.Error(err.Error())
				return
			}
		}
	}
	return
}

func ParseIPTest(db *sql.DB, resultLine string) (err error) {
	ipSet := strings.Split(resultLine, ",")
	for _, ip := range ipSet {
		intIP := InetAtoN(ip)
		// 查找ip地理坐标
		selectGeoSQL := "select geoname_id from GeoLite2-City-Blocks-IPv4 where long_ip_start <= ? and long_ip_end >= ?"
		var geoID sql.NullInt64
		err = db.QueryRow(selectGeoSQL).Scan(&geoID)
		if err != nil {
			log.Warning(err.Error())
			return
		}
		if !geoID.Valid {
			return
		}
		// 直接更新结果
		replaceIPSQL := "replace into `ip`(`id`, `ip`, `lnglat_id`) values (?, ?, ?)"
		_, err = db.Exec(replaceIPSQL, intIP, ip, geoID)
		if err != nil {
			log.Error(err.Error())
			return
		}
	}
}
