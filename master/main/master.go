package main

import (
	"flag"
	"fmt"
	"github.com/staryjie/crontab/master"
	"runtime"
	"time"
)

var (
	configFile string // 配置文件路径
)

// 解析命令行参数
func initArgs() {
	// master -config master.json
	flag.StringVar(&configFile, "config", "./master.json", "Enter program configuration file")
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
	if err = master.InitConfig(configFile); err != nil {
		return
	}

	// 任务管理器
	if err = master.InitJobMgr(); err != nil {
		goto ERR
	}

	// 启动API HTTP服务
	if err = master.InitApiServer(); err != nil {
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
