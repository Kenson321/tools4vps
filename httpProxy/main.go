package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

// 转发https
func handleTunneling(w http.ResponseWriter, r *http.Request) {
	//连接目标主机
	dest_conn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	//给客户端一个应答，既可以表明代理连接成功，又可以触发客户端继续发送新的消息从而两边开始通讯？
	w.WriteHeader(http.StatusOK)

	//劫持原连接
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	client_conn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}

	//透传，相互转发
	var wg sync.WaitGroup
	wg.Add(2)
	go transfer(dest_conn, nil, client_conn, &wg)
	go transfer(client_conn, nil, dest_conn, &wg)
	wg.Wait()
	client_conn.Close()
	dest_conn.Close()
}

func transfer(dest io.WriteCloser, log io.WriteCloser, source io.ReadCloser, wg *sync.WaitGroup) {
	//	w := io.MultiWriter(dest, log)
	//	io.Copy(w, source)
	io.Copy(dest, source)
	wg.Done()
}

// 转发http
func handleHTTP(w http.ResponseWriter, req *http.Request) {
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	io.Copy(w, resp.Body)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func runHttpsProxy(cert, key, port string) {
	fmt.Println("监听端口："+port)            

	server := &http.Server{
		Addr: ":" + port,
		Handler: http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				fmt.Printf("HandlerFunc收到请求，方法： %s，目标地址：%s，源地址：%s，请求：%s\n", r.Method, r.Host, r.RemoteAddr, r.RequestURI)
				if r.Method == http.MethodConnect {
					fmt.Println("代理https请求")    
					handleTunneling(w, r)
				} else {
					fmt.Println("代理http请求")    
					handleHTTP(w, r)
				}
			}),
		// Disable HTTP/2.
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	if len(cert) == 0 || len(key) == 0 {
		fmt.Println("启动http代理服务")    
		log.Fatal(server.ListenAndServe())
	} else {
		fmt.Println("启动https代理服务")    
		log.Fatal(server.ListenAndServeTLS(cert, key))
	}
}

func main() {
	fmt.Println(`支持代理http和https
通过http代理，远程服务器设置http地址：
export http_proxy="http://地址:端口"
export https_proxy="http://地址:端口"

通过https代理，指定证书和密钥启动https代理，远程服务器设置https地址：
export http_proxy="https://地址:端口"
export https_proxy="https://地址:端口"
`)

	cert := flag.String("cert", "", "path to cert file")
	key := flag.String("key", "", "path to key file")
	port := flag.String("port", "54321", "port")
	flag.Parse()

	runHttpsProxy(*cert, *key, *port)
}
