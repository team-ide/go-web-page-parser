package downloader

import (
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	defaultThreadNumber = 10
)

func Parser(url string, dir string, option *Option) (err error) {
	if option == nil {
		option = &Option{}
	}
	if option.ThreadNumber <= 0 {
		option.ThreadNumber = defaultThreadNumber
	}
	if dir == "" {
		dir = "./"
	}
	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}

	parser := &PageParser{
		Option:    option,
		url:       url,
		dir:       dir,
		numLocker: &sync.Mutex{},
	}

	err = parser.parse()
	if err != nil {
		return
	}
	return
}

type Option struct {
	ThreadNumber int
}

type PageParser struct {
	url string
	dir string
	*Option
	parsedUrls   []string
	responseList chan *streamResponse
	isStop       bool
	num          int
	numLocker    sync.Locker
}

type streamResponse struct {
	*http.Response
	url       string
	dir       string
	fileName  string
	err       error
	startTime time.Time
	endTime   time.Time
	isEnd     bool
}

func (this_ *PageParser) numR(num int) {
	this_.numLocker.Lock()
	defer this_.numLocker.Unlock()

	this_.num += num

}

func (this_ *PageParser) parse() (err error) {

	this_.responseList = make(chan *streamResponse, this_.ThreadNumber)
	var waitGroupForStop sync.WaitGroup
	var urlParsed = false
	go func() {
		for {
			response := <-this_.responseList
			fmt.Println("chan find response:", response)
			if response == nil {
				if urlParsed {
					break
				}
			}
			_ = this_.download(response)
		}
		waitGroupForStop.Done()

	}()
	waitGroupForStop.Add(1)
	fmt.Println("parse start url:", this_.url, " , dir:", this_.dir)
	err = this_.parseUrl(this_.url, this_.dir, false)
	fmt.Println("parse end url:", this_.url, " , dir:", this_.dir)
	urlParsed = true
	this_.responseList <- nil

	waitGroupForStop.Wait()

	return
}

func (this_ *PageParser) parseUrl(urlStr string, dir string, isFile bool) (err error) {
	if ContainsString(this_.parsedUrls, urlStr) >= 0 {
		return
	}
	this_.parsedUrls = append(this_.parsedUrls, urlStr)
	urlStr, _ = url.QueryUnescape(urlStr)
	fmt.Println("parseUrl url:", urlStr, " , dir:", dir)

	res, err := http.Get(urlStr)
	if err != nil {
		return
	}

	if isFile {
		response := &streamResponse{
			url:      urlStr,
			dir:      dir,
			Response: res,
		}

		this_.numR(1)
		this_.responseList <- response

		return
	}
	defer func() { _ = res.Body.Close() }()
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return
	}

	selection := doc.Find("a")
	selection.Each(func(i int, selection *goquery.Selection) {
		href, has := selection.Attr("href")
		if !has {
			return
		}
		if href == "" || href == "/" || href == "./" || href == "../" {
			return
		}
		if strings.Contains(href, "?") {
			return
		}
		if strings.HasPrefix(href, "/") {
			return
		}
		fmt.Println("selection url:", urlStr, " , href:", href)
		childrenUrl := urlStr
		if !strings.HasSuffix(href, "/") {
			childrenUrl += "/"
		}
		childrenUrl += href
		var e error
		if strings.HasSuffix(href, "/") {
			childrenDir := dir + href
			e = this_.parseUrl(childrenUrl, childrenDir, false)
		} else {
			e = this_.parseUrl(childrenUrl, dir, true)
		}
		if e != nil {
			err = e
		}

	})

	return
}

func (this_ *PageParser) download(response *streamResponse) (err error) {
	if response == nil {
		return
	}
	response.startTime = time.Now()
	defer func() {
		_ = response.Body.Close()
		if e := recover(); e != nil {
			err = errors.New(fmt.Sprint(e))
		}

		response.err = err
		response.endTime = time.Now()
		response.isEnd = true
		this_.numR(-1)
		go func() {
			this_.responseList <- nil
		}()
		fmt.Println("download end dir:", response.dir, " , fileName:", response.fileName, " , error:", err)

	}()

	urlStr, _ := url.QueryUnescape(response.url)
	r, err := url.Parse(urlStr)
	if err != nil {
		return
	}
	pathInfo := r.Path

	exist, err := PathExists(response.dir)
	if err != nil {
		return
	}
	if !exist {
		err = os.MkdirAll(response.dir, 0777)
		if err != nil {
			return
		}
	}

	ss := strings.Split(pathInfo, "/")
	response.fileName = ss[len(ss)-1]
	filePath := response.dir + response.fileName

	fmt.Println("download start dir:", response.dir, " , fileName:", response.fileName)

	f, err := os.Create(filePath)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()

	_, err = io.Copy(f, response.Body)
	if err != nil {
		return
	}
	fmt.Println("download success dir:", response.dir, " , fileName:", response.fileName)
	return

}

func ContainsString(array []string, val string) (index int) {
	index = -1
	for i := 0; i < len(array); i++ {
		if array[i] == val {
			index = i
			return
		}
	}
	return
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
