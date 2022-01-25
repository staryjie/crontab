package main

import (
	"flag"
	"fmt"
	"github.com/staryjie/crontab/worker"
	"runtime"
	"time"
)

var (
	configFile string // 配置文件路径
)

// 解析命令行参数
func initArgs() {
	// worker -config worker.json  生产环境建议填写绝对路径，因为你无法控制用户从哪里启动你的程序
	flag.StringVar(&configFile, "config", "/Users/staryjie/go/src/github.com/staryjie/crontab/worker/main/worker.json", "Enter program configuration file")
	flag.Parse()
}

// 解析线程数
func initEnv() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	var (
		err error
	)
	// 初始化命令行参数
	initArgs()

	// 初始化线程
	initEnv()

	// 加载配置
	if err = worker.InitConfig(configFile); err != nil {
		goto ERR
	}

	// 服务注册
	if err = worker.InitRegister(); err != nil {
		goto ERR
	}

	// 启动日志协程
	if err = worker.InitLogSink(); err != nil {
		goto ERR
	}

	// 启动执行器
	if err = worker.InitExcutor(); err != nil {
		goto ERR
	}

	// 启动调度器
	if err = worker.InitScheduler(); err != nil {
		goto ERR
	}

	// 初始化任务管理器
	if err = worker.InitJobMgr(); err != nil {
		goto ERR
	}

	// 正常退出
	// 测试使用，保证主进程不退出
	for {
		time.Sleep(1 * time.Second)
	}
	return
ERR:
	fmt.Println(err)
}
