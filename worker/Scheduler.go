package worker

import (
	"fmt"
	"github.com/staryjie/crontab/common"
	"strings"
	"time"
)

// 任务调度
type Scheduler struct {
	jobEventChan      chan *common.JobEvent              // Etcd任务事件队列
	jobPlanTable      map[string]*common.JobSchedulePlan // 任务调度计划表
	jobExecutingTable map[string]*common.JobExecuteInfo  // 任务执行表
	jobResultChan     chan *common.JobExecuteResult      // 任务执行结果队列
}

var (
	G_scheduler *Scheduler
)

// 处理任务事件
func (scheduler *Scheduler) handlerJobEvent(jobEvent *common.JobEvent) {
	var (
		jobSchedulePlan *common.JobSchedulePlan
		jobExisted      bool
		err             error
	)
	switch jobEvent.EventType {
	case common.JOB_EVENT_SAVE: // 更新事件
		if jobSchedulePlan, err = common.BuildJobSchedulePlan(jobEvent.Job); err != nil {
			return
		}
		// 加入到任务调度计划表
		scheduler.jobPlanTable[jobEvent.Job.Name] = jobSchedulePlan
	case common.JOV_EVENT_DELETE: // 删除事件
		if jobSchedulePlan, jobExisted = scheduler.jobPlanTable[jobEvent.Job.Name]; jobExisted {
			delete(scheduler.jobPlanTable, jobEvent.Job.Name)
		}
	}
}

// 尝试执行任务
func (scheduler *Scheduler) TryStartJob(jobPlan *common.JobSchedulePlan) {
	// 调度和执行时两件事
	var (
		jobExecuteInfo *common.JobExecuteInfo
		jobExecuting   bool
	)
	// 任务执行可能要很久，但是调度很频繁，比如1分钟调度60次，单次执行要1分钟，那么只有一次能够执行，防止并发执行

	// 如果任务正在执行，则跳过本次调度
	if jobExecuteInfo, jobExecuting = scheduler.jobExecutingTable[jobPlan.Job.Name]; jobExecuting {
		fmt.Println("任务正在执行中，跳过本次调度", jobPlan.Job.Name)
		return
	}

	// 构建执行状态信息
	jobExecuteInfo = common.BuildJobExecuteInfo(jobPlan)

	// 保存执行状态
	scheduler.jobExecutingTable[jobPlan.Job.Name] = jobExecuteInfo

	// 执行任务
	fmt.Println("执行任务:", jobExecuteInfo.Job.Name, jobExecuteInfo.PlanTime, jobExecuteInfo.RealTime)
	// TODO: 执行任务,执行完成从执行表中删除该任务
	G_executor.ExecuteJob(jobExecuteInfo)
}

// 重新计算任务调度状态
func (scheduler *Scheduler) TrySchedule() (scheduleAfer time.Duration) {
	var (
		jobPlan  *common.JobSchedulePlan
		now      time.Time
		nearTime *time.Time
	)

	// 如果任务表为空，睡眠1秒
	if len(scheduler.jobPlanTable) == 0 {
		scheduleAfer = 1 * time.Second
		return
	}

	// 1.遍历所有任务
	now = time.Now()
	for _, jobPlan = range scheduler.jobPlanTable {
		if jobPlan.NextTime.Before(now) || jobPlan.NextTime.Equal(now) {
			// 2.过期的任务马上执行
			// 尝试执行任务  // 上一个任务可能还在执行中
			scheduler.TryStartJob(jobPlan)
			fmt.Println(time.Now().Format("2006-01-02 15:04:05"), "执行任务:", jobPlan.Job.Name)
			// 更新下一次调度时间
			jobPlan.NextTime = jobPlan.Expr.Next(now)
		}
		// 3.计算最近一个要过期的任务，然后精确Sleep
		if nearTime == nil || jobPlan.NextTime.Before(*nearTime) {
			nearTime = &jobPlan.NextTime
		}
	}
	// 精准睡眠
	scheduleAfer = (*nearTime).Sub(now)
	return
}

// 处理任务执行结果
func (scheduler *Scheduler) handlerJobResult(jobResult *common.JobExecuteResult) {
	// 删除任务执行表中的该任务
	delete(scheduler.jobExecutingTable, jobResult.ExecuteInfo.Job.Name)

	fmt.Println("任务执行完成", jobResult.ExecuteInfo.Job.Name, strings.TrimSpace(string(jobResult.OutPut)), jobResult.Err)
}

// 调度协程
func (scheduler *Scheduler) scheduleLoop() {
	var (
		jobEvent      *common.JobEvent
		scheduleAfter time.Duration
		scheduleTimer *time.Timer
		jobResult     *common.JobExecuteResult
	)

	// 初始化计算任务调度状态执行任务
	scheduleAfter = scheduler.TrySchedule()

	// 调度的延迟定时器
	scheduleTimer = time.NewTimer(scheduleAfter)

	// 定时任务
	for {
		select {
		case jobEvent = <-scheduler.jobEventChan: // 监听任务变化事件
			// 对内存中的任务列表做增删改查
			scheduler.handlerJobEvent(jobEvent)
		case <-scheduleTimer.C: // 最近的任务到期
		case jobResult = <-scheduler.jobResultChan: // 监听任务执行结果
			scheduler.handlerJobResult(jobResult) // 处理任务执行结果
		}
		// 调度一次任务
		scheduleAfter = scheduler.TrySchedule()
		// 重置定时器
		scheduleTimer.Reset(scheduleAfter)
	}
}

// 推送任务变化事件
func (scheduler *Scheduler) PushJobEvent(jobEvent *common.JobEvent) {
	scheduler.jobEventChan <- jobEvent
}

// 初始化调度器
func InitScheduler() (err error) {
	G_scheduler = &Scheduler{
		jobEventChan:      make(chan *common.JobEvent, 1000),
		jobPlanTable:      make(map[string]*common.JobSchedulePlan),
		jobExecutingTable: make(map[string]*common.JobExecuteInfo),
		jobResultChan:     make(chan *common.JobExecuteResult, 1000),
	}
	go G_scheduler.scheduleLoop()
	return
}

// 任务执行回调函数,回传任务执行结果
func (scheduler *Scheduler) PushJobResult(jobResult *common.JobExecuteResult) {
	scheduler.jobResultChan <- jobResult
}
