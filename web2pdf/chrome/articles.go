package chrome

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"web2pdf/log"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/utils"
)

var Chrome = `C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`

// 获取url内的网页内容
func Visit(url string) (content string) {
	l := launcher.New().Headless(false) //打开浏览器，以便存在一些交互的操作
	_, err := os.Stat(Chrome)
	if err != nil {
		log.Error("未指定或未找到chrome执行程序：", Chrome)
	} else {
		l.Bin(Chrome)
	}
	cc := l.MustLaunch()
	browser := rod.New().ControlURL(cc).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage(url)

	done := make(chan bool)
	go func() {
		page.MustWaitStable()
		done <- true
	}()
	select {
	case <-done:
		log.Info("打开网页成功：", url)
	case <-time.After(time.Second * 30):
		log.Error("打开网页超时：", url)
	}

	time.Sleep(time.Second * 5) //方便交互操作

	title := ""
	if len(title) < 1 {
		exists, el, _ := page.HasX(`//*[@id="activity-name"]`)
		if exists {
			title = el.MustText()
		}
	}
	if len(title) < 1 {
		exists, el, _ := page.HasX(`//h1`)
		if exists {
			title = el.MustText()
		}
	}
	if len(title) < 1 {
		log.Error("获取标题元素失败", url)
		title = "noname"
	}
	title = strings.ReplaceAll(title, " ", "_")
	title = strings.ReplaceAll(title, "|", "_")
	title = strings.ReplaceAll(title, `:`, "_")
	title = strings.ReplaceAll(title, `"`, "_")
	title = strings.ReplaceAll(title, `?`, "_")
	title = strings.ReplaceAll(title, `*`, "_")
	title = strings.ReplaceAll(title, `/`, "_")
	title = strings.ReplaceAll(title, `\`, "_")
	title = strings.ReplaceAll(title, `<`, "_")
	title = strings.ReplaceAll(title, `>`, "_")
	log.Info("格式化后的标题：", title)

	baseDir := filepath.Join(".", title)
	os.MkdirAll(baseDir, os.ModeDir|os.ModePerm)

	//markdown
	mdfile, err := os.Create(filepath.Join(baseDir, title+".md"))
	if err != nil {
		log.Error("生成文件失败", err)
	}
	defer mdfile.Close()
	fmt.Fprintln(mdfile, title)
	fmt.Fprintln(mdfile, url)
	fmt.Fprintln(mdfile, "")
	el := page.MustElementX("/html")
	els := page.MustElementsX("//div[@id='js_content']/*")
	if len(els) > 0 {
		el = els[0]
	}
	resource := filepath.Join(baseDir, "resource")
	os.MkdirAll(resource, os.ModeDir|os.ModePerm)
	deepVisit(el, mdfile, resource, title)

	//txt
	txtfile, err := os.Create(filepath.Join(baseDir, title+".txt"))
	if err != nil {
		log.Error("生成文件失败", err)
	}
	defer txtfile.Close()
	el = page.MustElementX("/html")
	content = el.MustText()
	content = strings.Join(strings.Fields(content), " ")
	fmt.Fprintln(txtfile, content)

	//png
	page.MustScrollScreenshotPage(filepath.Join(baseDir, title+".png"))

	//pdf
	page.MustPDF(filepath.Join(baseDir, title+".pdf"))

	return
}

var repeat string

// 深度遍历，以支持获取图片和保持顺序
// 文字保存在f所代表的markdown文件中
// 图片保存在dir目录下
// title为目录相对路径名，用于在markdown文档中引用图片
func deepVisit(e *rod.Element, f *os.File, dir string, title string) {
	log.Debug(e.String())

	text := e.MustText()
	text = strings.TrimSpace(text)
	if len(text) > 0 && !strings.Contains(repeat, text) {
		log.Info("文字：", text)
		fmt.Fprintln(f, text)
		repeat = text
	}

	if strings.Contains(e.String(), "<img") {
		image(e, f, dir)
	}

	ne, err := e.ElementX("*")
	if err != nil {
		//return
	} else {
		deepVisit(ne, f, dir, title)
	}

	ne, err = e.Next()
	if err != nil {
		//return
	} else {
		deepVisit(ne, f, dir, title)
	}
}

func image(e *rod.Element, f *os.File, dir string) {
	var b0, b1, b2 []byte

	src, _ := e.Attribute("src")
	if src == nil {
		src = new(string)
	}
	log.Info("图片：", *src)

	if strings.HasPrefix(*src, "http") {
		//tp=webp 替换为 tp=nowebp
		//		if strings.Contains(newSrc, "tp=webp") {
		//			newSrc = strings.ReplaceAll(newSrc, "tp=webp", "tp=nowebp")
		//			log.Info("图片新地址：", newSrc)
		//		} else {
		b1 = e.MustResource()
		//		}
	}

	dataSrc, _ := e.Attribute("data-src")
	if dataSrc == nil {
		dataSrc = new(string)
	}
	log.Info("图片：", *dataSrc)
	if strings.Contains(*dataSrc, "tp=webp") {
		tsrc := strings.ReplaceAll(*dataSrc, "tp=webp", "tp=nowebp")
		dataSrc = &tsrc
		log.Info("图片新地址：", *dataSrc)
	}

	if strings.HasPrefix(*dataSrc, "https") {
		req, err := http.NewRequest("GET", *dataSrc, nil)
		if err != nil {
			log.Error("http发送失败：", err)
		} else {
			tls11Transport := &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			}
			client := &http.Client{
				Transport: tls11Transport,
			}
			res, err := client.Do(req)
			if err != nil {
				log.Error("http发送失败：", err)
			} else {
				defer res.Body.Close()
				b2, err = ioutil.ReadAll(res.Body)
				if err != nil {
					log.Error("http读取结果失败：", err)
				}
			}
		}
	} else if strings.HasPrefix(*dataSrc, "http") {
		res, err := http.Get(*dataSrc)
		if err != nil {
			log.Error("http发送失败：", err)
		} else {
			defer res.Body.Close()
			b2, err = ioutil.ReadAll(res.Body)
			if err != nil {
				log.Error("http读取结果失败：", err)
			}
		}
	}

	if len(b1) > len(b2) {
		b0 = b1
	} else {
		b0 = b2
	}
	if len(b0) > 0 {
		rand.Seed(time.Now().UnixNano())
		i := rand.Int31()
		imgF := filepath.Join(dir, fmt.Sprintf("%d.jpg", i))
		err := utils.OutputFile(imgF, b0)
		if err != nil {
			log.Error("生成图片失败：", err)
		} else {
			log.Info("生成图片：", imgF)
		}
		fmt.Fprintf(f, "![%d](.\\resource\\%d.jpg)\n", i, i)
		fmt.Fprintf(f, "%s\n", *src)
		fmt.Fprintf(f, "%s\n", *dataSrc)
	}
}
