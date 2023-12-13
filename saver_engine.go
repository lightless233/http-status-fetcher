package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"sync"
)

// SaverEngine 存储结果到文件
type SaverEngine struct {
	mainWG *sync.WaitGroup
	// waitGroup  *sync.WaitGroup
	resultChan *chan HTTPResult
}

// NewSaverEngine 创建一个新的 SaverEngine
func NewSaverEngine(mainWG *sync.WaitGroup, resultChan *chan HTTPResult) *SaverEngine {
	return &SaverEngine{
		mainWG, resultChan,
	}
}

// Run 启动 saverEngine
func (engine *SaverEngine) Run() {
	defer engine.mainWG.Done()
	engine.worker()
	sugarLogger.Info("SaverEngine stop.")
}

// worker
func (engine *SaverEngine) worker() {

	fp, err := os.OpenFile(appConfig.OutputFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE|os.O_TRUNC, 0666)
	defer func() {
		_ = fp.Close()
	}()
	if err != nil {
		sugarLogger.Fatalf("Can't open output file to write, filename: %s, error: %+v", appConfig.OutputFile, err)
		os.Exit(1)
	}
	writer := csv.NewWriter(fp)

	for {
		task, opened := <-*engine.resultChan
		if appConfig.Debug {
			sugarLogger.Debugf("Task: %+v, opened: %t", task, opened)
		}
		if !opened {
			break
		}

		// line := fmt.Sprintf(`"%d","%s","%s","%s"`, task.StatusCode, task.URL, task.Title, task.RedirectURL)
		writer.Write([]string{strconv.Itoa(task.StatusCode), task.URL, task.Title, task.RedirectURL})
		writer.Flush()

		sugarLogger.Debugf(fmt.Sprintf(`"%d","%s","%s","%s"`, task.StatusCode, task.URL, task.Title, task.RedirectURL))
	}
}
