// Copyright (C) AlexWoo(Wu Jie) wj19840501@gmail.com
//
// RTC Transport layer

package rtclib

import (
	"net/http"

	"github.com/gorilla/websocket"
)

type JSIPConfig struct {
	Realm string
}

type JSIPStack struct {
	config     JSIPConfig
	jsipHandle func(jsip *JSIP)
	log        *Log
	location   string
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  64 * 1024,
	WriteBufferSize: 64 * 1024,
	CheckOrigin:     wsCheckOrigin,
}

var jstack *JSIPStack

var transports = make(map[string]*websocket.Conn)

func InitJSIPStack(h func(jsip *JSIP), log *Log, location string) *JSIPStack {
	jstack = &JSIPStack{
		jsipHandle: h,
		log:        log,
		location:   location,
	}

	return jstack
}

func writemsg(conn *websocket.Conn, jsip *JSIP) {
	// TODO Error when sending msg
	conn.WriteJSON(jsip.RawMsg)
}

func readloop(name string, conn *websocket.Conn) {
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			jstack.log.LogError("RTC read message error, %v", err)
			break
		}

		if !RecvJSIPMsg(conn, data) {
			break
		}
	}

	conn.Close()

	delete(transports, name)
}

func wsCheckOrigin(r *http.Request) bool {
	return true
}

func RTCServer(w http.ResponseWriter, req *http.Request) {
	userid := req.URL.Query().Get("userid")
	if userid == "" {
		jstack.log.LogError("Miss userid")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		jstack.log.LogError("Create Websocket server failed, %v", err)
		return
	}

	transports[userid] = conn

	readloop(userid, conn)
}

func RTCClient(target string) *websocket.Conn {
	conn := transports[target]
	if conn != nil {
		return conn
	}

	dialer := websocket.DefaultDialer
	url := "http://" + target + jstack.location + "?userid=" + jstack.config.Realm
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		jstack.log.LogError("Connect to %s failed", url)
		return nil
	}

	transports[target] = conn

	go readloop(target, conn)

	return conn
}
