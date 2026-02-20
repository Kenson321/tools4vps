package main

import (
	"fmt"
	"net/http"
)

func main() {
	fmt.Println("监听54321端口，可以使用wget http://127.0.0.1:54321/下载当前目录下的文件")

	mux := http.NewServeMux()

	//如果文件目录下有index.html，则会直接作为页面
	//否则就显示文件目录结构
	files := http.FileServer(http.Dir(`.`))
	mux.Handle("/", http.StripPrefix("/", files))

	server := &http.Server{
		Addr:    "0.0.0.0:54321",
		Handler: mux,
	}

	server.ListenAndServe()
}
