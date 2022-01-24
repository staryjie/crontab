package worker

import (
	"context"
	"fmt"
	"github.com/staryjie/crontab/common"
	"math/rand"
	"os/exec"
	"time"
)

// 任务执行器
type Executor struct {
}

var (
	G_executor *Executor
)

// 执行一个任务
func (executor *Executor) ExecuteJob(info *common.JobExecuteInfo) {
	// 通过协程并发执行任务
	go func() {
		var (
			cmd     *exec.Cmd
			err     error
			output  []byte
			result  *common.JobExecuteResult
			jobLock *JobLock
		)

		// 任务执行结果
		result = &common.JobExecuteResult{
			ExecuteInfo: info,
			OutPut:      make([]byte, 0),
		}

		// TODO: 获取分布式锁，获取到锁才可以执行任务
		// 初始化锁
		jobLock = G_jobMgr.CreateJobLock(info.Job.Name)

		// 抢锁
		// 任务开始时间
		result.StartTime = time.Now()
		// 先随机睡眠0-1秒，保证每个客户端都能够抢到锁
		time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
		err = jobLock.TryLock()
		defer jobLock.Unlock() // 释放锁

		if err != nil { // 上锁失败
			fmt.Println(time.Now().Format("2006-01-02 15:04:05"), "抢锁失败", info.Job.Name)
			result.Err = err
			result.EndTime = time.Now()
		} else { // 抢锁成功，开始执行任务
			// 重置任务启动时间
			result.StartTime = time.Now()

			// 执行shell命令
			cmd = exec.CommandContext(context.TODO(), "/bin/bash", "-c", info.Job.Command)

			// 捕获输出或者异常
			output, err = cmd.CombinedOutput()

			// 任务结束时间
			result.EndTime = time.Now()

			result.OutPut = output
			result.Err = err
		}
		// 任务执行完成，把执行结果返回给Scheduler,Scheduler将该任务从jobExecutingTable中删除
		G_scheduler.PushJobResult(result)
	}()
}

// 初始化执行器
func InitExcutor() (err error) {
	G_executor = &Executor{}
	return
}
