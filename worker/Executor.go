package worker

import (
	"context"
	"github.com/staryjie/crontab/common"
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
			cmd    *exec.Cmd
			err    error
			output []byte
			result *common.JobExecuteResult
		)

		// 任务执行结果
		result = &common.JobExecuteResult{
			ExecuteInfo: info,
			OutPut:      make([]byte, 0),
		}

		// 任务开始时间
		result.StartTime = time.Now()

		// 执行shell命令
		cmd = exec.CommandContext(context.TODO(), "/bin/bash", "-c", info.Job.Command)

		// 捕获输出或者异常
		output, err = cmd.CombinedOutput()

		// 任务结束时间
		result.EndTime = time.Now()

		result.OutPut = output
		result.Err = err

		// 任务执行完成，把执行结果返回给Scheduler,Scheduler将该任务从jobExecutingTable中删除
		G_scheduler.PushJobResult(result)
	}()
}

// 初始化执行器
func InitExcutor() (err error) {
	G_executor = &Executor{}
	return
}
