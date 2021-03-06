package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"strconv"
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

// port-scan
type portScanPort struct {
	Protocol string `json:"protocol"`
	PortID   string `json:"portID"`
	State    string `json:"state"`
	Service  string `json:"service"`
	Product  string `json:"product"`
}

type portScanHost struct {
	Address string         `json:"address"`
	Ports   []portScanPort `json:"ports"`
}

func InetAtoN(ip string) int64 {
	ret := big.NewInt(0)
	ret.SetBytes(net.ParseIP(ip).To4())
	return ret.Int64()
}

func findIP(db *sql.DB, ip int64) (flag bool, err error) {
	findSQL := "select `id` from `ip` where id=?"
	var temp sql.NullInt64
	err = db.QueryRow(findSQL, ip).Scan(&temp)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		} else {
			return
		}
	}
	if temp.Valid {
		return true, nil
	} else {
		return false, nil
	}
}

func findGeoID(db *sql.DB, ip int64) (id int64, err error) {
	selectGeoSQL := "select `geoname_id` from `GeoLite2-City-Blocks-IPv4` " +
		"use index (`geolite2-city-blocks-ipv4_long_ip_start_index`,`geolite2-city-blocks-ipv4_long_ip_end_index`) " +
		"where `long_ip_start` <= ? and `long_ip_end` >= ?"
	var geoID sql.NullInt64
	err = db.QueryRow(selectGeoSQL, ip, ip).Scan(&geoID)
	if err != nil {
		log.Warning(err.Error())
		return
	}
	if !geoID.Valid {
		return -1, nil
	}
	return geoID.Int64, nil
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
	var updateParam []string
	var insertParam []string
	insertParamFormat := "(%d, '%s', %d, 1)"
	for _, ip := range ipSet {
		intIP := InetAtoN(ip)
		// insert or update
		flag, err := findIP(db, intIP)
		if err != nil {
			log.Error(err.Error())
			return err
		}
		if flag {
			updateParam = append(updateParam, strconv.FormatInt(intIP, 10))
		} else {
			// 查找ip地理坐标
			geoID, err := findGeoID(db, intIP)
			if err != nil {
				log.Error(err.Error())
				return err
			}
			insertParam = append(insertParam, fmt.Sprintf(insertParamFormat, intIP, ip, geoID))
		}
	}
	if len(updateParam) > 0 {
		updateSQL := "update `ip` set `ip_test_flag`=1 where id in (%s)"
		_, err = db.Exec(fmt.Sprintf(updateSQL, strings.Join(updateParam, ",")))
	}
	if err != nil {
		log.Error(err.Error())
		return err
	}
	if len(insertParam) > 0 {
		insertSQL := "insert into `ip`(`id`, `ip`, `lnglat_id`, `ip_test_flag`) values "
		_, err = db.Exec(insertSQL + strings.Join(insertParam, ","))
	}
	if err != nil {
		log.Error(err.Error())
	}
	return
}

func parsePortScan(db *sql.DB, resultLine string) (err error) {
	result := new([]portScanHost)
	err = json.Unmarshal([]byte(resultLine), result)
	if err != nil {
		fmt.Print(err.Error())
		return
	}
	var updateParam []string
	var insertParam []string
	insertParamFormat := "(%d, %s, '%s', '%s', '%s')"
	for _, h := range *result {
		intIP := InetAtoN(h.Address)
		updateParam = append(updateParam, strconv.FormatInt(intIP, 10))
		for _, p := range h.Ports {
			insertParam = append(
				insertParam,
				fmt.Sprintf(insertParamFormat, intIP, p.PortID, p.Protocol, p.Service, p.Product),
			)
		}
	}
	if len(updateParam) > 0 {
		updateSQL := "update `ip` set `port_scan_flag`=1 where id in (%s)"
		_, err = db.Exec(fmt.Sprintf(updateSQL, strings.Join(updateParam, ",")))
	}
	if err != nil {
		log.Error(err.Error())
		return err
	}
	if len(insertParam) > 0 {
		insertSQL := "insert into `port`(`ip_id`, `port`, `protocol`, `service`, `product`) values "
		_, err = db.Exec(insertSQL + strings.Join(insertParam, ","))
	}
	if err != nil {
		log.Error(err.Error())
	}
	return
}

func Url2Domain(url string) (domain string) {
	if strings.Contains(url, "http://") {
		return strings.Split(url[7:], "/")[0]
	} else if strings.Contains(url, "https://") {
		return strings.Split(url[8:], "/")[0]
	} else {
		return url
	}
}

func findUrl(db *sql.DB, url string) (id int, err error) {
	res := md5.Sum([]byte(url))
	hashKey := int64(binary.BigEndian.Uint64(res[0:8]))
	selectSQL := "select id from page use index (`page_domain_hash_index`) where domain_hash = ? and domain = ?"
	var idTemp sql.NullInt64
	err = db.QueryRow(selectSQL, hashKey, url).Scan(&idTemp)
	if err != nil && err != sql.ErrNoRows {
		log.Warning(err.Error())
		return
	}
	if err == sql.ErrNoRows || !idTemp.Valid {
		return -1, nil
	}
	id = int(idTemp.Int64)
	return
}

func parsePageCrawl(db *sql.DB, resultLine string) (err error) {
	point := strings.Index(resultLine, ",")
	url := Url2Domain(resultLine[0:point])
	htmlBase64 := resultLine[point+1:]
	// 查id
	id, err := findUrl(db, url)
	if err != nil || id == -1 {
		log.Warning(err.Error())
		return
	}
	dir := "/root/pages/" + strconv.Itoa(id/10000) + "/"
	htmlPath := dir + strconv.Itoa(id) + ".html"
	// 更新文件路径
	updateSQL := "update page set html_path = ? where id = ?"
	_, err = db.Exec(updateSQL, htmlPath, id)
	if err != nil {
		log.Warning(err.Error())
		return
	}
	// 写文件
	err = os.MkdirAll(dir, 0777)
	if err != nil && !os.IsExist(err) {
		log.Warning(err.Error())
		return
	}
	err = nil
	htmlBytes, err := base64.StdEncoding.DecodeString(htmlBase64)
	if err != nil {
		log.Warning(err.Error())
		return
	}
	html := string(htmlBytes)
	err = ioutil.WriteFile(htmlPath, []byte(html), 0644)
	return
}

func parseUrlCrawl(db *sql.DB, resultLine string) (err error) {
	if resultLine == "" {
		return nil
	}
	urls := strings.Split(resultLine, ",")
	insertFormat := "('%s', %d)"
	var insertValues []string
	for _, url := range urls {
		id, err := findUrl(db, url)
		if err != nil {
			log.Error(err.Error())
			return err
		}
		if id == -1 {
			res := md5.Sum([]byte(url))
			hashKey := int64(binary.BigEndian.Uint64(res[0:8]))
			insertValues = append(insertValues, fmt.Sprintf(insertFormat, url, hashKey))
		}
	}
	if len(insertValues) > 0 {
		insertSQL := "insert into `page`(`domain`, `domain_hash`) values "
		_, err = db.Exec(insertSQL + strings.Join(insertValues, ","))
	}
	return
}

func findDomain2IP(db *sql.DB, domain string, ipStr string) (isExisted bool, err error) {
	res := md5.Sum([]byte(domain))
	hashKey := int64(binary.BigEndian.Uint64(res[0:8]))
	ipInt := InetAtoN(ipStr)
	selectSQL := "select id from domain_ip use index(domain_ip_domain_hash_index, domain_ip_ip_id_index) " +
		"where ip_id = ? and domain_hash = ? and domain = ?"
	var idTemp sql.NullInt64
	err = db.QueryRow(selectSQL, ipInt, hashKey, domain).Scan(&idTemp)
	if err != nil && err != sql.ErrNoRows {
		log.Warning(err.Error())
		return
	}
	if err == sql.ErrNoRows || !idTemp.Valid {
		return false, nil
	}
	return true, nil
}

func parseDnssecure(db *sql.DB, resultLine string) (err error) {
	log.Debug(resultLine)
	results := strings.Split(resultLine, "\n")
	insertFormat := "('%s', %d, %d)"
	var insertValues []string
	for _, r := range results {
		ls := strings.Split(r, ";")
		if len(ls) < 6 {
			continue
		}
		domain := ls[2]
		as := strings.Split(strings.Split(ls[4], "/")[0], "+")
		res := md5.Sum([]byte(domain))
		hashKey := int64(binary.BigEndian.Uint64(res[0:8]))
		for _, a := range as {
			isExist, err := findDomain2IP(db, domain, a)
			if err != nil {
				log.Warning(err.Error())
				continue
			}
			if !isExist {
				ipInt := InetAtoN(a)
				insertValues = append(insertValues, fmt.Sprintf(insertFormat, domain, hashKey, ipInt))
			}
		}
	}
	if len(insertValues) > 0 {
		insertSQL := "insert into `domain_ip`(`domain`, `domain_hash`, `ip_id`) values "
		_, err = db.Exec(insertSQL + strings.Join(insertValues, ","))
	}
	return nil
}
