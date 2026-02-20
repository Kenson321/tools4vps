package main

import (
	"path/filepath"

	"sendMail/log"
	"sendMail/mail"
	"sendMail/props"
)

func main() {

	//日志开关
	log.SetDebug(props.Ppt.Debug, filepath.Join(props.Ppt.WorkDir, props.Ppt.LogFN))

	mail.Send()
}
