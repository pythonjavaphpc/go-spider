package main

/*
	macr0@vip.qq.com


*/
import (
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/widuu/goini"
	//"goini"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type TwebTree struct {
	Val      string
	Parent   *TwebTree
	Children map[string]*TwebTree
}

type SeTWebTree struct {
	Val      string
	Children map[string]*TwebTree
}

type TfileStruct struct {
	Turl string
	data string
	mget bool
}

var (
	mycfg               map[string]string
	blocks              []string
	WebTree             *TwebTree
	scanD               map[string]int = map[string]int{}
	Thread_n            int            = 100
	Thread_now          int            = 0
	ScanDeep            int            = 100
	SMode               int            = 0
	scookie             string         = ""
	ttf                 bool           = true
	web2file            bool           = false
	web2fileThread_n    int            = 100
	web2fileThread_now  int            = 0
	web2fileThread_chan chan bool
	fileStruct          chan TfileStruct
)

func md5str(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil)) // 输出加密结果
}

func newNode(v string) *TwebTree {
	return &TwebTree{Val: v, Children: make(map[string]*TwebTree)}
}

func (this *TwebTree) SetParent(p *TwebTree) {
	this.Parent = p
}

func (this *TwebTree) ExistChildren(key string) bool {
	if this.Children[key] != nil {
		return true
	} else {
		return false
	}
}

func (this *TwebTree) SetChildren(key string, p *TwebTree) *TwebTree {
	if !this.ExistChildren(key) {
		this.Children[key] = newNode(key)
		this.Parent = p
	}
	return this.Children[key]
}

func addWebTree(url string) {
	tmpRoot := strings.Split(url, "//")
	//log.Println(len(tmpRoot))
	if len(tmpRoot) == 2 {
		webPathtmp := strings.Split(tmpRoot[1], "?")
		//log.Println("webPathtmp", len(webPathtmp))
		if (len(webPathtmp) == 1) || (len(webPathtmp) == 2) {
			webPath := strings.Split(webPathtmp[0], "/")
			tmpNode := WebTree
			if len(webPath) >= 2 {
				for i := 0; i < len(webPath); i++ {
					if webPath[i] != "" {
						tmpNode = tmpNode.SetChildren(webPath[i], tmpNode)
					}
				}
			}
		}
	}
}

func echoWebTree(sp string, node *TwebTree) {
	i1 := 0
	i2 := len(node.Children)
	for key, value := range node.Children {
		i1++
		fmt.Println(sp+"+-", key)
		//}
		if len(value.Children) > 0 {
			if i1 == i2 {
				echoWebTree(sp+"    ", value)
			} else {
				echoWebTree(sp+"|   ", value)
			}
		}
	}
}

func ToSWebTree(sp string, node *TwebTree) string {
	OutputTree := ""
	i1 := 0
	i2 := len(node.Children)
	for key, value := range node.Children {
		i1++
		OutputTree = OutputTree + "\n" + sp + "+- " + key
		if len(value.Children) > 0 {
			if i1 == i2 {
				OutputTree = OutputTree + ToSWebTree(sp+"    ", value)
			} else {
				OutputTree = OutputTree + ToSWebTree(sp+"|   ", value)
			}

		}
	}
	return OutputTree
}

func webget(url, path, fname string) {
	client := &http.Client{}
	body := ""
	reqest, err := http.NewRequest("GET", url, nil)

	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		//os.Exit(0)
		web2fileThread_chan <- true
		return
	}

	reqest.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	//reqest.Header.Add("Accept-Encoding", "gzip, deflate")
	reqest.Header.Add("Accept-Language", "zh-cn,zh;q=0.8,en-us;q=0.5,en;q=0.3")
	reqest.Header.Add("Connection", "keep-alive")
	//reqest.Header.Add("Host", "login.sina.com.cn")
	//reqest.Header.Add("Referer", "http://weibo.com/")
	reqest.Header.Add("User-Agent", "Mozilla/5.0 (compatible; MSIE 10.0; Windows NT 6.2; Trident/6.0)")

	response, err := client.Do(reqest)
	defer response.Body.Close()

	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		//os.Exit(0)
		web2fileThread_chan <- true
		return
	}

	if response.StatusCode == 200 {
		//var body string

		switch response.Header.Get("Content-Encoding") {
		case "gzip":
			reader, _ := gzip.NewReader(response.Body)
			for {
				buf := make([]byte, 1024)
				n, err := reader.Read(buf)

				if err != nil && err != io.EOF {
					panic(err)
				}

				if n == 0 {
					break
				}
				body += string(buf)
			}
		default:
			bodyByte, _ := ioutil.ReadAll(response.Body)
			body = string(bodyByte)
		}
	}
	//return body
	os.MkdirAll(path, 0700)
	fout, err := os.Create(path + fname)
	defer fout.Close()
	if err == nil {
		fout.WriteString(body)
		//web2fileThread_chan <- true
		//fout.Write(data)
	}
	//fout.Close()
	web2fileThread_chan <- true
}

func web2f(Turl, data string, mget bool) {
	url := Turl
	Turl = strings.Split(Turl, "://")[1]
	ttp := strings.Split(Turl, "?")
	Turl = ttp[0]
	fname := ""
	if len(ttp) > 1 {
		fname = "_" + ttp[1] + ".html"
	}
	Turl = strings.Split(Turl, "#")[0]
	Turl = strings.Replace(Turl, "//", "/", -1)
	Turl = strings.Replace(Turl, "/", "\\", -1)
	if strings.HasSuffix(Turl, "\\") {
		fname = "index" + fname
	} else {
		tp := strings.Split(Turl, "\\")
		//fmt.Println("------------", len(tp), tp)
		tabc := ""
		i := 0
		for ; i < len(tp)-1; i++ {
			tabc = tabc + tp[i] + "\\"
		}
		fname = tp[i] + fname
		//fmt.Println("------------", tabc)
		Turl = tabc
	}

	if fname == "index" {
		fname = "index.html"
	}

	if _, err := os.Stat(Turl + fname); err != nil {
		//time.Sleep(100000000) //100ms
		if mget {
			if web2fileThread_now <= web2fileThread_n {
				go webget(url, Turl, fname)
				web2fileThread_now++
			} else {
				if <-web2fileThread_chan {
					go webget(url, Turl, fname)
				}
			}
			fmt.Println("资源->" + url)
		} else {
			os.MkdirAll(Turl, 0700)
			fout, err := os.Create(Turl + fname)
			//defer fout.Close()
			if err == nil {
				fout.WriteString(data)
				//fout.Write(data)
			}
			fout.Close()
			//fmt.Println("-----", Turl, fname, url /*, data*/)
		}
	}
}

func web2fCtrl() {
	for tmpData := range fileStruct {
		web2f(tmpData.Turl, tmpData.data, tmpData.mget)
	}
}

func httpSpider(Turl string, isTread bool) {
	if isTread {
		Thread_now++
	}
	var bodystr string
	time.Sleep(100000000) //100ms
	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				deadline := time.Now().Add(25 * time.Second)
				c, err := net.DialTimeout(netw, addr, time.Second*20)
				if err != nil {
					return nil, err
				}
				c.SetDeadline(deadline)
				return c, nil
			},
		},
	}
	reqest, err := http.NewRequest("GET", Turl, nil)
	if err == nil {
		reqest.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		reqest.Header.Set("Accept-Charset", "GBK,utf-8;q=0.7,*;q=0.3")
		//reqest.Header.Set("Accept-Encoding", "gzip,deflate,sdch")
		reqest.Header.Set("Accept-Language", "zh-CN,zh;q=0.8")
		reqest.Header.Set("Cache-Control", "max-age=0")
		reqest.Header.Set("Connection", "keep-alive")
		reqest.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MSIE 10.0; Windows NT 6.2; Trident/6.0)")
		if scookie != "" {
			reqest.Header.Set("Cookie", scookie)
		}

		goct := true
		tmpRoot := strings.Split(Turl, "//")
		if len(tmpRoot) >= 2 {
			webRoot := tmpRoot[0] + "//" + strings.Split(tmpRoot[1], "/")[0]

			response, err := client.Do(reqest)
			if err != nil {
				goct = false
			}
			if goct {
				if response.StatusCode == 200 {
					switch response.Header.Get("Content-Encoding") {
					case "gzip":
						reader, _ := gzip.NewReader(response.Body)
						for {
							buf := make([]byte, 1024)
							n, err := reader.Read(buf)

							if err != nil && err != io.EOF {
								panic(err)
							}

							if n == 0 {
								break
							}
							bodystr += string(buf)
						}
					default:
						body, _ := ioutil.ReadAll(response.Body)
						bodystr = string(body)
					}

					if web2file {
						//web2f(Turl, bodystr, false)
						fileStruct <- TfileStruct{Turl, bodystr, false}
					}
					reg := regexp.MustCompile(`(?i:href|src)[ |\t]*=(.+?)(?i:#|"|'| |\t|>)`) //`(?i:href|src)=(?:\"|'| |)(.|\n)+?(?:\"|'| |\>)`)
					//(?<=(?i:href|src)=(?:\"|'| |))(.*?)(?=(?:\"|'| |>))
					//reg := regexp.MustCompile(`(?U)(?i:href|src)=.+(?:\"|'| |>)`)
					//`(?U)\b.+\b`
					//`(?U)(?i:href|src)=.+(?:\"|'| |>)`
					urls := reg.FindAllString(bodystr, -1)
					for _, v := range urls {
						v = strings.Replace(v, "\"", "", -1)
						v = strings.Replace(v, "'", "", -1)
						v = strings.Replace(v, " ", "", -1)
						v = strings.Replace(v, "\t", "", -1)
						v = strings.Replace(v, ">", "", -1)
						v = strings.Replace(v, "href=", "", 1)
						v = strings.Replace(v, "src=", "", 1)
						v = strings.Split(v, "#")[0]
						if strings.HasPrefix(v, "//") {
							v = "http:" + v
						}
						v_tmp := v
						v = strings.Split(v, "?")[0]

						isBlockde := false
						for _, value := range blocks {
							if value != "" {
								if strings.Contains(v_tmp, value) {
									isBlockde = true
									break
								}
							}
						}

						if !isBlockde {
							if (v != "") && (v != "/") && (v != "\\") && (v != "(") && (!strings.HasPrefix(v, "<")) && (!strings.HasPrefix(v, "mailto:")) && (!strings.HasPrefix(v, "data:")) && (!strings.HasPrefix(v, "#")) && (strings.ToLower(v) != "about:blank") && (!strings.HasPrefix(strings.ToLower(v), "javascript")) && (!strings.Contains(strings.ToLower(v_tmp), "&nbsp;")) && (!strings.Contains(strings.ToLower(v_tmp), "&amp;")) && (!strings.Contains(strings.ToLower(v_tmp), "&gt;")) && (!strings.Contains(strings.ToLower(v_tmp), "&lt;")) && (strings.Count(v_tmp, "?") <= 1) && (!strings.HasPrefix(v, "//")) {
								if strings.HasPrefix(v, "/") {
									v = webRoot + v
									v_tmp = webRoot + v_tmp
								} else {
									if !strings.HasPrefix(v, "http") {
										//fmt.Println("----"+webRoot, v, v_tmp)
										thipath := strings.Split(Turl, "://")[1]
										thipath = strings.Split(thipath, "?")[0]
										thipath = strings.Split(thipath, "#")[0]
										thipath = strings.Replace(thipath, "//", "/", -1)
										tp := strings.Split(thipath, "/")
										//fmt.Println("------------", len(tp), tp)
										thipath = ""
										if !strings.HasSuffix(thipath, "/") {
											for i := 1; i < len(tp)-1; i++ {
												thipath = thipath + tp[i] + "/"
											}
										} else {
											for i := 1; i < len(tp); i++ {
												thipath = thipath + tp[i] + "/"
											}
										}
										v = webRoot + "/" + thipath + v
										v_tmp = webRoot + "/" + thipath + v_tmp
									}
								}
								//fmt.Println("----"+v, v_tmp)

								if strings.HasPrefix(v, webRoot) || SMode == 1 {
									addWebTree(v)
									TTT := strings.Split(v, ".")
									if len(TTT) > 2 {
										if scanD[v_tmp] > 0 {
											//ok
										} else {

											if !strings.Contains("js,txt,log,jpg,ico,png,gif,css,swf,htc,rar,zip,doc", TTT[len(TTT)-1]) {
												scanD[v_tmp] = 100

												fmt.Println(Thread_now, Thread_n, len(scanD), response.StatusCode, v_tmp)
												if Thread_n > 0 {
													Thread_n--
													go httpSpider(v_tmp, true)
												} else {
													httpSpider(v_tmp, false)
												}
												if len(scanD) > ScanDeep {
													if isTread {
														Thread_now--
													}
													return
												}
											} else {
												if web2file {
													//web2f(v_tmp, bodystr, true)
													fileStruct <- TfileStruct{v_tmp, bodystr, true}
													//fmt.Println(v_tmp)
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	if isTread {
		Thread_now--
	}
}

func toFile() {
	userFile := "Spider.log"
	tmpRoot := strings.Split(mycfg["url"], "//")
	webRoot := tmpRoot[0] + "//" + strings.Split(tmpRoot[1], "/")[0]
	i := 0
	for {
		time.Sleep(1000000000) //1000ms
		i++
		if i >= 240 {
			i = 0
			fout, err := os.Create(userFile)
			//defer fout.Close()
			if err == nil {
				fout.WriteString("------------------[WebTree]----------------------\n")
				fout.WriteString(webRoot + "\n/" + ToSWebTree("", WebTree))
				fout.WriteString("\n----------------[UrlList]----------------------")
				fout.WriteString("\n                 Len: " + strconv.Itoa(len(scanD)) + "\n")
				// for key_url, _ := range scanD {
				// 	fout.WriteString(key_url + "\n")
				// }
				fout.WriteString("\n-----------------------------------------------")
			}
			fout.Close()
		}
	}
}

func TimeToFile(webRoot string) {
	time.Sleep(10000000000) //10s
	webRoot = strings.Replace(webRoot, ":", "_", -1)
	os.MkdirAll(webRoot, 0700)
	if ttf {
		userFile := webRoot + "\\Spider.log"
		fout, err := os.Create(userFile)
		defer fout.Close()
		if err == nil {
			fout.WriteString("------------------[WebTree]----------------------\n")
			fout.WriteString(webRoot + "\n/" + ToSWebTree("", WebTree))
			fout.WriteString("\n----------------[UrlList]----------------------")
			fout.WriteString("\n                 Len: " + strconv.Itoa(len(scanD)) + "\n")
			for key_url, _ := range scanD {
				fout.WriteString(key_url + "\n")
			}
			fout.WriteString("\n-----------------------------------------------")
		}
	}
}

func main() {
	mycfg = make(map[string]string)
	fileStruct = make(chan TfileStruct)
	web2fileThread_chan = make(chan bool)
	//blocks = make(map[int]string)
	//mycfg["url"] = "http://www.baidu.com"
	//cfg.Load("conn.cfg", mycfg)
	conf := goini.SetConfig("config.ini")
	mycfg["url"] = conf.GetValue("Config", "Url")
	mycfg["thread"] = conf.GetValue("Config", "Thread")
	mycfg["deep"] = conf.GetValue("Config", "Deep")
	mycfg["mode"] = conf.GetValue("Config", "Mode")
	mycfg["block"] = conf.GetValue("Config", "Block")
	mycfg["tofile"] = conf.GetValue("Config", "Tofile")
	scookie = conf.GetValue("Config", "Cookie")
	WebTree = newNode("/")

	TTi, err := strconv.Atoi(mycfg["thread"])
	if err == nil {
		Thread_n = TTi
	} else {
		Thread_n = 100
	}

	web2fileThread_n = Thread_n

	TTi, err = strconv.Atoi(mycfg["deep"])
	if err == nil {
		ScanDeep = TTi
	} else {
		ScanDeep = 100
	}

	TTi, err = strconv.Atoi(mycfg["mode"])
	if err == nil {
		SMode = TTi
	} else {
		SMode = 0
	}

	/*switch len(os.Args) {
	case 2:
		mycfg["url"] = os.Args[1]
	case 3:
		mycfg["url"] = os.Args[1]
		scookie = os.Args[2]
	}*/

	//fmt.Println(mycfg["block"]) //////////////////////////////////////////////////////////////
	blocks = strings.Split(mycfg["block"], ",")

	if mycfg["tofile"] == "1" {
		web2file = true
	} else {
		web2file = false
	}

	log.Println("[" + mycfg["deep"] + "][" + mycfg["block"] + "][" + mycfg["mode"] + "][" + mycfg["url"] + "][" + scookie + "]")

	//fmt.Println("-->" + webget("http://apps.bdimg.com/developer/static/1408011145/social-widget/js/vote.js"))
	//time.Sleep(100 * 1000000000) //10s
	tmpRoot := strings.Split(mycfg["url"], "//")
	webRoot := strings.Split(tmpRoot[1], "/")[0]
	go TimeToFile(webRoot)
	go web2fCtrl()
	webRoot = tmpRoot[0] + "//" + webRoot
	go httpSpider(mycfg["url"], true)
	for i := 0; len(scanD) <= ScanDeep; /*&& (i <= 10)*/ {
		time.Sleep(1000000000) //1000ms
		if Thread_n == 0 {
			i++
		} else {
			i = 0
		}
		if Thread_now <= 0 {
			break
		}
	}

	log.Println("\nscanD:", len(scanD))
	fmt.Println(webRoot + "\n/")

	userFile := "Spider.log"
	fout, err := os.Create(userFile)
	defer fout.Close()
	if err == nil {
		fout.WriteString("------------------[WebTree]----------------------\n")
		fout.WriteString(webRoot + "\n/" + ToSWebTree("", WebTree))
		fout.WriteString("\n----------------[UrlList]----------------------")
		fout.WriteString("\n                 Len: " + strconv.Itoa(len(scanD)) + "\n")
		for key_url, _ := range scanD {
			fout.WriteString(key_url + "\n")
		}
		fout.WriteString("\n-----------------------------------------------")
	}
	echoWebTree("", WebTree)
}
