package main

import (
	"bufio"
	"io"
	"os"
	"strings"
	"sync"
)

// TaskBuilderEngine 生产任务的引擎
type TaskBuilderEngine struct {

	// 存放主线程的 wg
	mainWG *sync.WaitGroup

	// 任务队列
	taskChan *chan string
}

// NewTaskBuilderEngine 创建一个新的 TaskBuilderEngine
func NewTaskBuilderEngine(mainWG *sync.WaitGroup, taskChan *chan string) *TaskBuilderEngine {
	return &TaskBuilderEngine{
		mainWG, taskChan,
	}
}

// Run 启动 TaskBuilder 引擎
func (engine *TaskBuilderEngine) Run() {
	defer func() {
		engine.mainWG.Done()
		// 结束的时候关闭任务队列，以确保 fetcherEngine 可以正常结束
		close(*engine.taskChan)
	}()
	engine.worker()
}

// worker
func (engine *TaskBuilderEngine) worker() {
	var successCount uint = 0

	if appConfig.Target != "" {
		// 通过命令行参数传递的
		targets := strings.Split(appConfig.Target, ",")
		for _, t := range targets {
			t = strings.TrimSpace(t)
			*engine.taskChan <- t
			successCount += 1
		}
		sugarLogger.Infof("%d tasks were successfully added.", successCount)
	} else if appConfig.InputFile != "" {
		// 通过文件输入指定
		fp, err := os.Open(appConfig.InputFile)
		defer func() { _ = fp.Close() }()
		if err != nil {
			sugarLogger.Errorf("Error when reading input file: %s, error: %+v", appConfig.InputFile, err)
			return
		}

		// 按行读文件
		br := bufio.NewReader(fp)
		for {
			line, err := br.ReadString('\n')
			if err != nil && err != io.EOF {
				sugarLogger.Errorf("Error when reading input file: %s, error: %+v", appConfig.InputFile, err)
				return
			}

			line = strings.TrimSpace(line)
			// TODO 校验域名是否合法
			*engine.taskChan <- line
			successCount += 1

			if err == io.EOF {
				break
			}
		}

		sugarLogger.Infof("%d tasks were successfully added.", successCount)
	} else {
		logger.Error("appConfig.Target and appConfig.InputFile cannot be empty at the same time.")
	}
}
