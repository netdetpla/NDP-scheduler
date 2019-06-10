package main

import (
	"net"
)

func udpServer(r *resource)  {
	addr, err := net.ResolveUDPAddr("udp", ":" + r.Port)
	if err != nil {
		log.Error(err.Error())
		return
	}
	listener, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Error(err.Error())
		return
	}
	log.Notice("UDP server for resource started.")
	data := make([]byte, 1024)
	for {
		n, remoteAddr, err := listener.ReadFromUDP(data)
		if err != nil {
			log.Warning(err.Error())
			continue
		}
		log.Infof("<%s> %s", remoteAddr, data[:n])
	}
}
