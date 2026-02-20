package main

import (
	"flag"
	"web2pdf/chrome"
	// "web2pdf/log"
)

func main() {
	url := flag.String("url", "https://www.baidu.com/", "url")
	flag.Parse()

	//日志开关
	//	log.SetDebug(true, "")

	chrome.Visit(*url)
}
