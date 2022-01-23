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
	// master -config master.json  生产环境建议填写绝对路径，因为你无法控制用户从哪里启动你的程序
	flag.StringVar(&configFile, "config", "/Users/staryjie/go/src/github.com/staryjie/crontab/master/main/master.json", "Enter program configuration file")
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
		goto ERR
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
	fmt.Printf("[%v] Crontab Server started ...\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("Please Visit http://127.0.0.1:%d/\n", master.G_Config.ApiPort)
	for {
		time.Sleep(1 * time.Second)
	}
	return
ERR:
	fmt.Println(err)
}
