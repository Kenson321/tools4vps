// https://github.com/cw1997/NATBypass
package main

import (
	"flag"
	"io"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const timeout = 5

var logPath *string

func main() {
	log.SetFlags(log.Ldate | log.Lmicroseconds)

	logPath = flag.String("lp", ``, "log path")
	p1 := flag.String("l1", ``, "listen port 1. input port")
	p2 := flag.String("l2", ``, "listen port 2, only if listen port 1 presents. input port")
	d1 := flag.String("d1", ``, "dial ip:port 1. input ip:port")
	d2 := flag.String("d2", ``, "dial ip:port 2, only if dial ip:port 1 presents. input ip:port")
	et := flag.Int("et", 0, "encrypt tunnel. input 1 or 2, default 0 means not encrypt")
	flag.String("h", ``, `help
	1、同时监听2个端口。当两个客户端主动连接上这两个监听端口之后，负责这两个端口间的数据转发
	"-l1 port1 -l2 port2" example: "main -l1 1997 -l2 2017"
	
	2、监听1个端口，连接1个主机。当监听端口接收到来自客户端的连接之后，主动连接主机，并负责端口和主机之间的数据转发
	"-l1 port1 -d1 ip:port2" example: "main -l1 1997 -d1 192.168.1.2:3389"
	
	3、连接2个主机。主动连接两个主机，连接成功之后，负责这两个主机之间的数据转发
	"-d1 ip1:port1 -d2 ip2:port2" example: "main -d1 127.0.0.1:3389 -d2 8.8.8.8:1997"
	
	支持通道加密，-et 2代表使用加解密连接通道2，对应另一方也要需要通过 -et 1进行加解密连接
	example:
	本地 "main -l1 1997 -d1 192.168.1.2:3389 -et 2"
	远程 "main -l1 3389 -d1 xxx.xxx.xxx.xxx:1998 -et 1"
	不存在同时两边加解密的情况，如此则直接不加解密即可

	注意！！！
	对于https的转发，注意本地需要修改/etc/hosts文件，增加"127.0.0.1 域名"，并用域名取代ip进行访问。
	因为https会检查证书中的域名是否匹配`)
	flag.Parse()

	if len(*p1) > 0 && len(*p2) > 0 {
		log.Println("start to listen port:", *p1, "and port:", *p2)
		port1 := checkPort(*p1)
		port2 := checkPort(*p2)
		port2port(port1, port2, *et)
	} else if len(*p1) > 0 && len(*d1) > 0 {
		log.Println("start to listen port:", *p1, "and dial address:", *d1)
		port := checkPort(*p1)
		remoteAddress := checkIp(*d1)
		port2host(port, remoteAddress, *et)
	} else if len(*d1) > 0 && len(*d2) > 0 {
		log.Println("start to dial address:", *d1, "and address:", *d2)
		address1 := checkIp(*d1)
		address2 := checkIp(*d2)
		host2host(address1, address2, *et)
	} else {
		flag.PrintDefaults()
	}
}

// 检查端口是否合法
func checkPort(port string) string {
	PortNum, err := strconv.Atoi(port)
	if err != nil {
		log.Fatalln(port, "port should be a number")
	}
	if PortNum < 1 || PortNum > 65535 {
		log.Fatalln(port, "port should be a number and the range is [1,65536)")
	}
	return port
}

// 检查ip:端口是否合法
func checkIp(address string) string {
	ipAndPort := strings.Split(address, ":")
	if len(ipAndPort) != 2 {
		log.Fatalln(address, "address error. should be a string like [ip:port]. ")
	}
	ip := ipAndPort[0]
	port := ipAndPort[1]
	checkPort(port)
	pattern := `^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$`
	ok, err := regexp.MatchString(pattern, ip)
	if err != nil || !ok {
		log.Fatalln(address, "ip error. ")
	}
	return address
}

// 不断监听发送到2个端口的连接，连接成功，则转发两个端口的内容
func port2port(port1 string, port2 string, et int) {
	listen1 := start_server("0.0.0.0:" + port1)
	defer listen1.Close()
	listen2 := start_server("0.0.0.0:" + port2)
	defer listen2.Close()
	for {
		ch1 := make(chan net.Conn, 1)
		ch2 := make(chan net.Conn, 1)
		go accept(listen1, ch1)
		go accept(listen2, ch2)
		conn1 := <-ch1
		conn2 := <-ch2

		forward(conn1, conn2, et)
	}
}

// 不断监听本地端口的连接，连接成功后通过net.Dial创建到远程主机的连接，然后转发两个连接的内容
func port2host(allowPort string, targetAddress string, et int) {
	server := start_server("0.0.0.0:" + allowPort)
	defer server.Close()
	for {
		ch1 := make(chan net.Conn, 1)
		go accept(server, ch1)
		conn := <-ch1

		//println(targetAddress)
		go func(targetAddress string) {
			ch2 := make(chan net.Conn, 1)
			go connect(targetAddress, ch2, 100)
			// TODO 如果100次都失败，下面这里也会被挂起，导致这个协程无法被释放
			target := <-ch2

			forward(conn, target, et)
		}(targetAddress)
	}
}

// 连接2个主机，连接成功后转发两个连接的内容
func host2host(address1, address2 string, et int) {
	ch1 := make(chan net.Conn, 1)
	ch2 := make(chan net.Conn, 1)
	for {
		go connect(address1, ch1, 0)
		go connect(address2, ch2, 0)
		host1 := <-ch1
		host2 := <-ch2

		forward(host1, host2, et)
	}
}

// 通过net.Dial连接主机
func connect(targetAddress string, ch chan net.Conn, max int) {
	for i := 0; i <= max; {
		log.Println("start connect host:[" + targetAddress + "]")
		target, err := net.Dial("tcp", targetAddress)
		if err != nil {
			log.Println("[x]", strconv.Itoa(i), "connect target address ["+targetAddress+"] faild. retry in ", timeout, "seconds. ")
			if target != nil {
				target.Close()
			}
			time.Sleep(timeout * time.Second)
			if max > 0 {
				i++
			}
			continue
		}
		ch <- target
		log.Println("connect target address [" + targetAddress + "] success.")
		break
	}
}

// 使用net.Listen监听端口
func start_server(address string) net.Listener {
	log.Println("try to start server on:[" + address + "]")
	server, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalln("listen address [" + address + "] faild.")
	}
	log.Println("start listen at address:[" + address + "]")
	return server
}

// 使用listener.Accept创建连接
func accept(listener net.Listener, ch chan net.Conn) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("[x]", "accept connect ["+conn.RemoteAddr().String()+"] faild.", err.Error())
			if conn != nil {
				conn.Close()
			}
			log.Println("retry in ", timeout, " seconds. ")
			time.Sleep(timeout * time.Second)
			continue
		}
		log.Println("accept a new client. remote address:[" + conn.RemoteAddr().String() + "], local address:[" + conn.LocalAddr().String() + "]")
		ch <- conn
		break
	}
}

// 完成conn2和conn1的转发。把conn2收到的内容发送到conn1，同时把conn1收到的内容发送到conn2
func forward(conn1 net.Conn, conn2 net.Conn, et int) {
	log.Printf("start transmit. [%s],[%s] <-> [%s],[%s] \n", conn1.LocalAddr().String(), conn1.RemoteAddr().String(), conn2.LocalAddr().String(), conn2.RemoteAddr().String())
	var wg sync.WaitGroup
	wg.Add(2)
	if et == 1 {
		go connCopy(conn1, conn2, &wg, CopyToEncrypt)
		go connCopy(conn2, conn1, &wg, CopyFromDecrypt)
	} else if et == 2 {
		go connCopy(conn1, conn2, &wg, CopyFromDecrypt)
		go connCopy(conn2, conn1, &wg, CopyToEncrypt)
	} else {
		go connCopy(conn1, conn2, &wg, io.Copy)
		go connCopy(conn2, conn1, &wg, io.Copy)
	}
	wg.Wait()
	conn1.Close()
	log.Println("close the connect at local:[" + conn1.LocalAddr().String() + "] and remote:[" + conn1.RemoteAddr().String() + "]")
	conn2.Close()
	log.Println("close the connect at local:[" + conn2.LocalAddr().String() + "] and remote:[" + conn2.RemoteAddr().String() + "]")
}

// 把conn2接收到的内容转发到conn1，并同时写入文本
func connCopy(dst net.Conn, src net.Conn, wg *sync.WaitGroup, f func(dst io.Writer, src io.Reader) (written int64, err error)) {
	logFile := openLog(dst.LocalAddr().String(), dst.RemoteAddr().String(), src.LocalAddr().String(), src.RemoteAddr().String())
	if logFile != nil {
		w := io.MultiWriter(dst, logFile)
		f(w, src)
	} else {
		f(dst, src)
	}
	wg.Done()
}

// 打开日志文件
func openLog(address1, address2, address3, address4 string) *os.File {
	if len(*logPath) == 0 {
		return nil
	}
	var logFileError error
	var logFile *os.File
	address1 = strings.Replace(address1, ":", "_", -1)
	address2 = strings.Replace(address2, ":", "_", -1)
	address3 = strings.Replace(address3, ":", "_", -1)
	address4 = strings.Replace(address4, ":", "_", -1)
	timeStr := time.Now().Format("2006_01_02_15_04_05") // "2006-01-02 15:04:05"
	logPath := *logPath + "/" + timeStr + "-" + address1 + "_" + address2 + "-" + address3 + "_" + address4 + ".log"
	logPath = strings.Replace(logPath, `\`, "/", -1)
	logPath = strings.Replace(logPath, "//", "/", -1)
	logFile, logFileError = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE, 0666)
	if logFileError != nil {
		log.Println("[x]", "log file path error.", logFileError.Error())
		return nil
	}
	log.Println("open log file success. path:", logPath)
	return logFile
}
