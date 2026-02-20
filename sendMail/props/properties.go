package props

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"sendMail/log"
)

const propFileName = "mail.properties"

func init() {
	initProps()
}

type Properties struct {
	Debug           bool     `json:"debug"`        //输出debug日志
	LogFN           string   `json:"logfn"`        //日志文件
	WorkDir         string   `json:"wdir"`         //工作目录，不指定时支持在主目录下动态创建，指定时则无需指定主目录
	MailUser        string   `json:"muser"`        //发送邮件的邮箱用户名
	MailPwd         string   `json:"mpwd"`         //发送邮件的邮箱用户密码
	MailSubj        string   `json:"msubj"`        //发送爬取结果邮件的主题，不指定时动态创建避免误中垃圾邮件判断
	MailMessage     string   `json:"mmsg"`         //
	MailTO          []string `json:"mto"`          //主送邮箱
	MailCC          []string `json:"mcc"`          //抄送邮箱
	MailBCC         []string `json:"mbcc"`         //暗送邮箱
	MailImages      []string `json:"mimages"`      //
	MailAttachments []string `json:"mattachments"` //
}

var Ppt *Properties

func initProps() {
	Ppt = &Properties{
		Debug:           true,
		LogFN:           `日志.log`,
		WorkDir:         `D:\`,
		MailUser:        "发送邮件的邮箱用户名",
		MailPwd:         "发送邮件的邮箱用户密码",
		MailSubj:        "主题",
		MailTO:          []string{"主送邮箱1", "主送邮箱2"},
		MailCC:          []string{"抄送邮箱1", "抄送邮箱2"},
		MailBCC:         []string{"暗送邮箱1", "暗送邮箱2"},
		MailImages:      []string{},
		MailAttachments: []string{`日志.log`},
	}

	jsonF, err := os.Open(propFileName)
	if err != nil {
		log.Error("打开配置文件失败，系统将自动生成新的配置文件，请修改参数后重新执行", propFileName, err)
		initPropFile()
		panic(err)
	}
	defer jsonF.Close()

	b, err := ioutil.ReadAll(jsonF)
	if err != nil {
		log.Error("读取文件失败：", err)
		panic(err)
	}

	err = json.Unmarshal(b, &Ppt)
	if err != nil {
		log.Error("解析json失败", err)
		panic(err)
	}
}

func initPropFile() {
	jsonF, err := os.Create(propFileName)
	if err != nil {
		log.Error("打开文件失败", err)
		return
	}
	defer jsonF.Close()

	b, _ := json.Marshal(Ppt)
	var bb bytes.Buffer
	json.Indent(&bb, b, "", "\t")

	fmt.Fprintf(jsonF, "%s", bb.String())
}
