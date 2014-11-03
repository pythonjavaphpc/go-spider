package main

/*
	by: macr0@vip.qq.com

conn.cfg:
	url=http://www.qq.com/
	Thread=500
	Deep=3000
	Mode=0                      0:This domain    1:All

*/
import (
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/jimlawless/cfg"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"runtime"
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

var (
	mycfg      map[string]string
	WebTree    *TwebTree
	scanD      map[string]int = map[string]int{}
	Thread_n   int            = 100
	Thread_now int            = 0
	ScanDeep   int            = 100
	SMode      int            = 0
	scookie    string         = ""
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
		reqest.Header.Set("Accept-Encoding", "deflate,sdch")
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
						v_tmp := v
						v = strings.Split(v, "?")[0]
						if (v != "") && (v != "/") && (v != "\\") && (v != "(") && (!strings.HasPrefix(v, "<")) && (!strings.HasPrefix(v, "#")) && (strings.ToLower(v) != "about:blank") && (!strings.HasPrefix(strings.ToLower(v), "javascript")) && (!strings.Contains(strings.ToLower(v), "&nbsp;")) && (!strings.Contains(strings.ToLower(v), "&gt;")) && (!strings.Contains(strings.ToLower(v), "&lt;")) && (strings.Count(v, "?") <= 1) && (!strings.HasPrefix(v, "//")) {
							if strings.HasPrefix(v, "/") {
								v = webRoot + v
								v_tmp = webRoot + v_tmp
							} else {
								if !strings.HasPrefix(v, "http") {
									v = webRoot + "/" + v
									v_tmp = webRoot + "/" + v_tmp
								}
							}

							if strings.HasPrefix(v, webRoot) || SMode == 1 {
								addWebTree(v)
								TTT := strings.Split(v, ".")
								if len(TTT) > 2 {
									if scanD[v_tmp] > 0 {
										//ok
									} else {

										if !strings.Contains("js,jpg,ico,png,gif,css,swf,htc,rar,zip,doc", TTT[len(TTT)-1]) {
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

func main() {
	mycfg = make(map[string]string)
	mycfg["url"] = "http://www.baidu.com"
	cfg.Load("conn.cfg", mycfg)
	WebTree = newNode("/")

	TTi, err := strconv.Atoi(mycfg["thread"])
	if err == nil {
		Thread_n = TTi
	} else {
		Thread_n = 100
	}

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

	switch len(os.Args) {
	case 2:
		mycfg["url"] = os.Args[1]
	case 3:
		mycfg["url"] = os.Args[1]
		scookie = os.Args[2]
	}

	log.Println(mycfg["deep"], ScanDeep, err, len(os.Args), os.Args)
	//fmt.Println(runtime.NumCPU()
	runtime.GOMAXPROCS(runtime.NumCPU())
	go toFile()
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

	tmpRoot := strings.Split(mycfg["url"], "//")
	webRoot := tmpRoot[0] + "//" + strings.Split(tmpRoot[1], "/")[0]
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
