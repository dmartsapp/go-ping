// @Description:
// @File:  pinger.go
// @Author: github.com/farhansabbir
// @Date: 2024-05-12 22:28

package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"time"
)

type Pinger struct {
	Destination        *net.IPAddr `json:"destination"`
	TTL                int         `json:"ttl"`
	NameResolveTimeout int         `json:"name_resolve_timeout"`
	Payload            string      `json:"payload"`
}

func NewPinger(destination_name string, resolvetimeout int) *Pinger {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(resolvetimeout))
	addr, err := net.DefaultResolver.LookupIPAddr(ctx, destination_name)
	if err != nil {
		log.Fatal("Unable to resolve destination name")
	}
	defer cancel()
	return &Pinger{
		Destination:        &addr[0],
		NameResolveTimeout: resolvetimeout,
	}
}

func (p *Pinger) ToString() string {
	str, _ := json.Marshal(p)
	return string(str)
}
