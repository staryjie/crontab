package master

import (
	"net"
	"net/http"
	"strconv"
	"time"
)

var (
	// 单例对象
	G_apiServer *ApiServer
)

// 任务的HTTP接口
type ApiServer struct {
	httpServer *http.Server
}

// 保存任务接口
func handleJobSave(w http.ResponseWriter, r *http.Request) {
	// 任务保存到etcd中
}

// 初始化服务
func InitApiServer() (err error) {
	var (
		mux        *http.ServeMux
		listener   net.Listener
		httpServer *http.Server
	)
	// 初始化路由
	mux = http.NewServeMux()
	mux.HandleFunc("/job/save", handleJobSave)

	// 启动HTTP监听
	if listener, err = net.Listen("tcp", ":" + strconv.Itoa(G_Config.ApiPort)); err != nil {
		return
	}

	// 创建一个HTTP服务
	httpServer = &http.Server{
		ReadTimeout:  time.Duration(G_Config.ApiReadTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(G_Config.ApiWriteTimeout) * time.Millisecond,
		Handler:      mux,
	}

	// 给单例赋值
	G_apiServer = &ApiServer{
		httpServer: httpServer,
	}

	// 协程启动服务端
	go httpServer.Serve(listener)

	return
}
