package main

import (
	"flag"
	"fmt"
	"github.com/ritterhou/stinger/core/codec"
	"github.com/ritterhou/stinger/core/network"
	localConf "github.com/ritterhou/stinger/local/conf"
	"github.com/ritterhou/stinger/local/http"
	"github.com/ritterhou/stinger/local/socks"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

var (
	confFile     string
	remoteServer string
	password     string
)

func init() {
	t := time.Now()
	now := t.Format("20060102150405")
	filename := fmt.Sprintf("stinger_local.%s.log", now)

	logFile, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalln("open log file failed", err)
	}
	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	flag.StringVar(&confFile, "c", "stinger_local.yaml", "Local configuration file.")

	localConf.LoadConf(confFile)
	conf := localConf.GetConf()

	pac := conf["pac"].(map[interface{}]interface{})
	pacPort := pac["port"].(int)
	global := pac["global"].(bool)
	domains := pac["domains"].([]interface{})

	localPort := conf["local_port"].(int)
	remoteServer = conf["server_address"].(string)

	pwd := conf["password"]
	switch v := pwd.(type) {
	case int:
		password = strconv.Itoa(v)
	case string:
		password = v
	default:
		log.Println("Unknown type ", v)
	}

	codec.SetKey(password)

	http.CreatePacFile(localPort, global, domains)
	go http.StartServer(pacPort)

	go socks.BandwidthTraffic()

	startProxyServer(localPort)
}

func startProxyServer(proxyPort int) {
	var l net.Listener
	var err error
	var host = "0.0.0.0:" + strconv.Itoa(proxyPort)

	l, err = net.Listen("tcp", host)
	if err != nil {
		log.Fatal("Error listening:", err)
	}
	defer l.Close()

	log.Println("Local ProxyServer working on " + host)
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println("Error accepting:", err)
			continue
		}
		go handlerSocks5(network.Connection{Conn: conn})
	}
}

func handlerSocks5(conn network.Connection) {
	err := socks.AuthSocks5(conn)
	if err != nil {
		log.Println(err)
		return
	}
	remoteConn, err := socks.ConnectRemote(conn, remoteServer, password)
	if err != nil {
		log.Println(err)
		return
	}

	//log.Printf("Connect success %s -> %s, %s => %s\n", conn.RemoteAddress(), conn.LocalAddress(), remoteConn.LocalAddress(), remoteConn.RemoteAddress())
	socks.HandlerSocks5Data(conn, remoteConn)
}
