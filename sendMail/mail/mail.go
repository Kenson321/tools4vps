package mail

import (
	"bytes"
	"crypto/tls"
	"html/template"
	"path/filepath"

	"sendMail/log"
	"sendMail/props"

	"gopkg.in/gomail.v2"
)

// 发送邮件
func Send() {

	message := `
<p>您好</p>

<p style="text-indent:2em">` + props.Ppt.MailMessage + `</P>
<p style="text-indent:2em">见附件</P>

<p style="text-indent:2em">祝好</P>
`
	images := []string{}
	attachments := []string{}

	nImages := len(props.Ppt.MailImages)
	if nImages > 0 {
		ct, err := template.New("mail").Parse(`
<p>您好</p>

<p style="text-indent:2em">` + props.Ppt.MailMessage + `</P>

{{range .}}
<img src="cid:{{.}}" />
{{end}}

<p style="text-indent:2em">见附件</P>

<p style="text-indent:2em">祝好</P>
`)
		if err != nil {
			log.Error("模板分析失败", err)
			return
		}
		msg := new(bytes.Buffer)
		err = ct.Execute(msg, props.Ppt.MailImages)
		if err != nil {
			log.Error("转换为html失败", err)
			return
		}
		message = msg.String()

		images = make([]string, nImages, nImages)
		for n, imageFN := range props.Ppt.MailImages {
			img := filepath.Join(props.Ppt.WorkDir, imageFN)
			images[n] = img
		}
	}

	nAttch := len(props.Ppt.MailAttachments)
	if nAttch > 0 {
		attachments = make([]string, nAttch, nAttch)
		for n, attchFN := range props.Ppt.MailAttachments {
			attch := filepath.Join(props.Ppt.WorkDir, attchFN)
			attachments[n] = attch
		}
	}

	send163(props.Ppt.MailUser, props.Ppt.MailPwd, props.Ppt.MailSubj, message, props.Ppt.MailTO, props.Ppt.MailCC, props.Ppt.MailBCC, images, attachments)
}

// go get -v gopkg.in/gomail.v2
func send163(userName, password, subj, message string, mailTo, mailCC, mailBCC, images, attachments []string) {
	// 163 邮箱：
	// SMTP 服务器地址：smtp.163.com（端口：25）
	host := "smtp.163.com"
	port := 25

	m := gomail.NewMessage()

	m.SetAddressHeader("From", userName, "信息搜集") // 增加发件人别名（支持中文）
	//	m.SetHeader("From", userName) // 发件人
	//	m.SetHeader("From", "WechatArticles"+"<"+userName+">") // 增加发件人别名（不支持中文）
	m.SetHeader("To", mailTo...)   // 收件人，可以多个收件人，但必须使用相同的 SMTP 连接
	m.SetHeader("Cc", mailCC...)   // 抄送，可以多个
	m.SetHeader("Bcc", mailBCC...) // 暗送，可以多个
	m.SetHeader("Subject", subj)   // 邮件主题

	// text/html 的意思是将文件的 content-type 设置为 text/html 的形式，浏览器在获取到这种文件时会自动调用html的解析器对文件进行相应的处理。
	// 可以通过 text/html 处理文本格式进行特殊处理，如换行、缩进、加粗等等
	m.SetBody("text/html", message)
	// text/plain的意思是将文件设置为纯文本的形式，浏览器在获取到这种文件时并不会对其进行处理
	// m.SetBody("text/plain", "纯文本")

	for _, img := range images {
		m.Embed(img) //图片
	}

	for _, att := range attachments {
		m.Attach(att) // 附件文件，可以是文件，照片，视频等等
	}

	d := gomail.NewDialer(
		host,
		port,
		userName,
		password,
	)
	// 关闭SSL协议认证
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	log.Info("发送邮件")
	err := d.DialAndSend(m)
	if err != nil {
		log.Error("发送邮件失败", err, subj)
	}
}
