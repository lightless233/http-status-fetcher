package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

var client = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
	Timeout: 12 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

var titlePattern = regexp.MustCompile(`<title>(?P<title>.+?)</title>`)

// FetcherEngine 获取 HTTP 信息的引擎
type FetcherEngine struct {
	mainWG    *sync.WaitGroup
	waitGroup *sync.WaitGroup

	taskChan   *chan string
	resultChan *chan HTTPResult
}

// NewFetcherEngine 创建一个新的 FetcherEngine
func NewFetcherEngine(mainWG *sync.WaitGroup, taskChan *chan string, resultChan *chan HTTPResult) *FetcherEngine {
	var waitGroup sync.WaitGroup
	return &FetcherEngine{
		mainWG:     mainWG,
		taskChan:   taskChan,
		resultChan: resultChan,
		waitGroup:  &waitGroup,
	}
}

// Run 启动 FetcherEngine
func (engine *FetcherEngine) Run() {
	defer func() {
		engine.mainWG.Done()
		close(*engine.resultChan)
	}()

	// 启动协程
	var i uint = 0
	for ; i < appConfig.TaskCount; i++ {
		engine.waitGroup.Add(1)
		go engine.worker(i)
	}

	// 等待协程结束
	engine.waitGroup.Wait()
	sugarLogger.Info("FetcherEngine stop.")
}

// worker 真正的工作函数
func (engine *FetcherEngine) worker(idx uint) {
	defer engine.waitGroup.Done()
	tag := fmt.Sprintf("[Fetcher-%d]", idx)

	sugarLogger.Debugf("%s start.", tag)
	for {
		task, opened := <-*engine.taskChan
		if appConfig.Debug {
			sugarLogger.Debugf("%s receive task: %s, opened: %t", tag, task, opened)
		}
		if !opened {
			break
		}

		// 拿到了 task，需要先格式化task
		// 1. 如果 task 自带协议(https: or http:)，就直接请求不做处理
		// 2. 如果是 //example.com ，则先补 https 再补 http
		// 3. 如果是 example.com 则先补 https:// 再补 http://
		urlBuffer := make([]string, 0, 2)
		task = strings.TrimSpace(task)
		if strings.HasPrefix(task, "https:") || strings.HasPrefix(task, "http:") {
			urlBuffer = append(urlBuffer, task)
		} else if strings.HasPrefix(task, "//") {
			urlBuffer = append(urlBuffer, fmt.Sprintf("https:%s", task))
			urlBuffer = append(urlBuffer, fmt.Sprintf("http:%s", task))
		} else {
			urlBuffer = append(urlBuffer, fmt.Sprintf("https://%s", task))
			urlBuffer = append(urlBuffer, fmt.Sprintf("http://%s", task))
		}
		sugarLogger.Debugf("%s url buffer: %+v", tag, urlBuffer)

		// 依次对 URL buffer 中的每个 URL 进行请求
		for _, url := range urlBuffer {
			res, err := makeRequest(url)
			if err != nil {
				sugarLogger.Warnf("%s Error when request URL %s, error: %+v", tag, url, err)
			} else {
				*engine.resultChan <- *res
				break
			}
		}

	}
	sugarLogger.Debugf("%s stop.", tag)
}

func makeRequest(url string) (*HTTPResult, error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	bContent, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	content := string(bContent[:])

	statusCode := response.StatusCode
	location := response.Header.Get("Location")

	// 从 content 中匹配出 title
	var title string
	p := titlePattern.FindStringSubmatch(content)
	if len(p) < 2 {
		title = ""
	} else {
		title = p[1]
	}

	return &HTTPResult{
		URL:         url,
		StatusCode:  statusCode,
		Title:       title,
		RedirectURL: location,
	}, nil
}
