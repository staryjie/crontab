package master

import (
	"encoding/json"
	"github.com/staryjie/crontab/common"
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
// POST job = {"name": "job1", "command": "echo hello", "cronExpr": "* * * * *"}
func handleJobSave(resp http.ResponseWriter, req *http.Request) {
	var (
		err     error
		postJob string
		job     common.Job
		oldJob  *common.Job
		bytes   []byte
	)
	// 任务保存到etcd中
	// 1. 解析POST表单
	if err = req.ParseForm(); err != nil {
		// 解析表单失败
		goto ERR
	}
	// 2.取表单中的job对象
	postJob = req.PostForm.Get("job")

	// 3.反序列化Job
	if err = json.Unmarshal([]byte(postJob), &job); err != nil {
		// 反序列化失败
		goto ERR
	}

	// 4.保存到Etcd
	if oldJob, err = G_jobMgr.SaveJob(&job); err != nil {
		goto ERR
	}

	// 5.返回正常应答
	if bytes, err = common.BuildResponse(0, "success", oldJob); err == nil { // 正常响应 err 应该为 nil
		resp.Write(bytes)
	}

	return
ERR:
	// 返回异常应答
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

// 删除任务接口
// POST /job/delete  name = job1
func handleJobDelete(resp http.ResponseWriter, req *http.Request) {
	var (
		name   string
		err    error
		oldJob *common.Job
		bytes  []byte
	)

	// 解析Form表单
	if err = req.ParseForm(); err != nil {
		goto ERR
	}

	// 获取到要删除的任务名称
	name = req.PostForm.Get("name")

	// 通过任务名去删除任务
	if oldJob, err = G_jobMgr.DeleteJob(name); err != nil {
		goto ERR
	}

	// 正常删除的应答
	if bytes, err = common.BuildResponse(0, "success", oldJob); err == nil {
		resp.Write(bytes)
	}
	return
ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

// 获取所有任务的列表
func handleJobList(resp http.ResponseWriter, req *http.Request) {
	var (
		jobList []*common.Job
		bytes   []byte
		err     error
	)

	// 获取任务列表
	if jobList, err = G_jobMgr.ListJobs(); err != nil {
		goto ERR
	}

	// 正常应答
	if bytes, err = common.BuildResponse(0, "success", jobList); err == nil {
		resp.Write(bytes)
	}
	return
ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

// 强杀任务
func handleJobKill(resp http.ResponseWriter, req *http.Request) {
	var (
		name  string
		bytes []byte
		err   error
	)

	// 解析POST表单
	if err = req.ParseForm(); err != nil {
		goto ERR
	}

	// 获取要杀死的任务
	name = req.PostForm.Get("name")

	// 杀死任务
	if err = G_jobMgr.KillJob(name); err != nil {
		goto ERR
	}

	// 正常响应
	if bytes, err = common.BuildResponse(0, "success", nil); err == nil {
		resp.Write(bytes)
	}
	return
ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

// 日志查询
func handleJobLog(resp http.ResponseWriter, req *http.Request) {
	var (
		err        error
		name       string // 任务名字
		skipParam  string // 从第几条开始
		limitParam string // 返回多少条
		skip       int
		limit      int
		logArr     []*common.JobLog
		bytes      []byte
	)

	// 解析GET参数
	if err = req.ParseForm(); err != nil {
		goto ERR
	}

	// 获取请求参数 /job/log?name=job10&skip=0&limit=10
	name = req.Form.Get("name")
	skipParam = req.Form.Get("skip")
	limitParam = req.Form.Get("limit")
	if skip, err = strconv.Atoi(skipParam); err != nil {
		skip = 0
	}
	if limit, err = strconv.Atoi(limitParam); err != nil {
		limit = 5
	}

	if logArr, err = G_logMgr.ListLog(name, skip, limit); err != nil {
		goto ERR
	}

	// 正常应答
	if bytes, err = common.BuildResponse(0, "success", logArr); err == nil {
		resp.Write(bytes)
	}
	return

ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		resp.Write(bytes)
	}
}

// 初始化服务
func InitApiServer() (err error) {
	var (
		mux           *http.ServeMux
		listener      net.Listener
		httpServer    *http.Server
		staticDir     http.Dir     // 静态文件根目录
		staticHandler http.Handler // 静态文件HTTP回调
	)
	// 初始化路由
	mux = http.NewServeMux()
	mux.HandleFunc("/job/save", handleJobSave)     // 保存任务
	mux.HandleFunc("/job/delete", handleJobDelete) // 删除任务
	mux.HandleFunc("/job/list", handleJobList)     // 获取所有任务
	mux.HandleFunc("/job/kill", handleJobKill)     // 强杀任务
	mux.HandleFunc("/job/log", handleJobLog)       // 日持查询

	// http支持静态文件路由
	staticDir = http.Dir(G_Config.WebRoot)
	staticHandler = http.FileServer(staticDir)
	mux.Handle("/", http.StripPrefix("/", staticHandler))

	// 启动HTTP监听
	if listener, err = net.Listen("tcp", ":"+strconv.Itoa(G_Config.ApiPort)); err != nil {
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
