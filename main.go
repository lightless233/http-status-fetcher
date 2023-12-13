package main

// 1. 如果输入的目标上没有指定端口号，默认 443，如果访问不通降级到 80
// 2. 输出字段：URL，status_code，title, redirect_url(如果有302、301等状态)

//var logger = GetSugar()

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"os"
	"runtime"
	"sync"
)

func main() {
	app := &cli.App{
		Usage:   "HTTP Status Fetcher",
		Action:  mainAction,
		Version: "0.1.0",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "target",
				Usage:       "Target domain",
				Destination: &appConfig.Target,
				Aliases:     []string{"t"},
			},
			&cli.StringFlag{
				Name:        "input-file",
				Usage:       "A file contains a list of domains to be scanned, one line per domain",
				Destination: &appConfig.InputFile,
				Aliases:     []string{"i"},
			},
			&cli.UintFlag{
				Name:        "task-count",
				Usage:       "Task count",
				Destination: &appConfig.TaskCount,
				Aliases:     []string{"n"},
				Value:       uint(2*runtime.NumCPU() + 1),
				DefaultText: "2 * CPU + 1",
			},
			&cli.StringFlag{
				Name:        "output",
				Usage:       "output filename",
				Aliases:     []string{"o"},
				Destination: &appConfig.OutputFile,
				Value:       "./out.txt",
			},
			&cli.BoolFlag{
				Name:        "debug",
				Usage:       "Debug mode",
				Value:       false,
				Destination: &appConfig.Debug,
			},
		},
		Before: func(context *cli.Context) error {
			// 初始化日志系统
			debug := context.Bool("debug")
			InitLogger(debug)

			return nil
		},
	}

	if err := app.Run(os.Args); nil != err {
		sugarLogger.Errorf("Error when run app. Error: %+v", err)
		os.Exit(1)
	}

}

func mainAction(c *cli.Context) error {
	sugarLogger.Infof("HTTPStatusFetcher start.")
	sugarLogger.Debugf("appConfig: %+v", appConfig)

	// 检查参数冲突
	if appConfig.InputFile != "" && appConfig.Target != "" {
		err := "the 'target' and 'input' parameters cannot be set at the same time"
		logger.Error(err)
		return fmt.Errorf(err)
	}
	if appConfig.InputFile == "" && appConfig.Target == "" {
		err := "the 'target' and 'input' cannot be empty at the same time"
		return fmt.Errorf(err)
	}

	// taskBuilderEngine -> taskChan -> fetcherEngine -> resultChan -> saverEngine
	// 任务队列
	taskChan := make(chan string, 64)
	// 结果队列
	resultChan := make(chan HTTPResult, 4)

	var waitGroup sync.WaitGroup

	// 启动 saverEngine
	saverEngine := NewSaverEngine(&waitGroup, &resultChan)
	waitGroup.Add(1)
	go saverEngine.Run()

	// 启动 fetcherEngine
	fetcherEngine := NewFetcherEngine(&waitGroup, &taskChan, &resultChan)
	waitGroup.Add(1)
	go fetcherEngine.Run()

	// 启动 taskBuilderEngine
	taskBuilderEngine := NewTaskBuilderEngine(&waitGroup, &taskChan)
	waitGroup.Add(1)
	go taskBuilderEngine.Run()

	waitGroup.Wait()
	sugarLogger.Infof("HTTPStatusFetcher end. Write result to file: %s", appConfig.OutputFile)

	return nil
}
