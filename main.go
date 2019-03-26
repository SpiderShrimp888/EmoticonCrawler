package main

import (
	"bytes"
	"flag"
	"fmt"
	"golang.org/x/net/html"
	"htmlquery"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	TARGET_URL       = "https://www.doutula.com/photo/list/"
	DEFAULT_MAX_PAGE = 100
	PIC_EXPR         = "//img[@class=\"img-responsive lazy image_dta\" and contains(@alt,\"%s\")]"
	PAGE_EXPR        = "//a[@class=\"page-link\"]//text()"
	TIME_FMT         = "2006年01月02日15时04分05秒001毫秒"
)

var (
	TotalCount = 0
	CountMutex = new(sync.Mutex)
	KeyFlag=flag.String("keyword","","search for the specific pictures")
)

func main() {
	flag.Parse()
	dirName := time.Now().Format(TIME_FMT)
	os.Mkdir(dirName, os.ModePerm)

	startTime := time.Now()

	maxPage := GetMaxPage()
	signalList := make([]chan int, maxPage)

	for index := 1; index <= maxPage; index++ {
		signalList[index-1] = make(chan int)

		root, _ := htmlquery.LoadURL(fmt.Sprintf("%s?page=%d", TARGET_URL, index))

		go func(myIndex int, node *html.Node) {
			picCount := 0

			htmlquery.FindEach(root,
				fmt.Sprintf(PIC_EXPR, *KeyFlag),
				func(index int, node *html.Node) {
					picUrlAttr := htmlquery.SelectAttr(node, "data-backup")
					picUrl := string([]rune(picUrlAttr)[:strings.LastIndex(picUrlAttr, "!dta")])

					strList := strings.Split(picUrl, ".")
					suffix := strList[len(strList)-1]

					func(url string) {
						res, _ := http.Get(url)
						picBytes, _ := ioutil.ReadAll(res.Body)

						fullFileName := fmt.Sprintf("%s/%s.%s",
							dirName,
							htmlquery.SelectAttr(node, "alt"),
							suffix)
						file, _ := os.Create(fullFileName)
						io.Copy(file, bytes.NewReader(picBytes))

						picCount++

						fmt.Println(fmt.Sprintf("%s.%s", htmlquery.SelectAttr(node, "alt"),
							suffix))
					}(picUrl)
				})

			signalList[myIndex-1] <- picCount
		}(index, root)
	}

	for _, count := range signalList {
		TotalCount += <-count
	}

	fmt.Println()
	fmt.Printf("已完成下载，本次共下载表情%d张，耗时:%s\r\n", TotalCount, time.Since(startTime).String())
	fmt.Printf("表情图片保存路径：%s/%s\r\n",GetRunPath(),dirName)
	fmt.Println("按任意键+回车退出...")
	var quitSignal string
	fmt.Scan(&quitSignal)
}

func GetMaxPage() int {
	maxPage := DEFAULT_MAX_PAGE

	root, _ := htmlquery.LoadURL(TARGET_URL)
	htmlquery.FindEach(root,
		PAGE_EXPR,
		func(index int, node *html.Node) {

			page, err := strconv.Atoi(node.Data)
			if err == nil && page > maxPage {
				maxPage = page
			}
		})

	return maxPage
}

func GetRunPath() string {
	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))

	return strings.Replace(dir, "\\", "/", -1)
}