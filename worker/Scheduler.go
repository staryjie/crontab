package worker

import (
	"fmt"
	"github.com/staryjie/crontab/common"
	"time"
)

type Scheduler struct {
	jobEventChan chan *common.JobEvent  // Etcd任务事件队列
	jobPlanTable map[string]*common.JobSchedulePlan  // 任务调度计划表
}

var (
	G_scheduler *Scheduler
)

// 处理任务事件
func (scheduler *Scheduler) HandlerJobEvent(jobEvent *common.JobEvent) {
	var (
		jobSchedulePlan *common.JobSchedulePlan
		jobExisted bool
		err error
	)
	switch jobEvent.EventType {
	case common.JOB_EVENT_SAVE:  // 更新事件
		if jobSchedulePlan, err = common.BuildJobSchedulePlan(jobEvent.Job); err != nil {
			return
		}
		// 加入到任务调度计划表
		scheduler.jobPlanTable[jobEvent.Job.Name] = jobSchedulePlan
	case common.JOV_EVENT_DELETE:  // 删除事件
		if jobSchedulePlan, jobExisted = scheduler.jobPlanTable[jobEvent.Job.Name]; jobExisted {
			delete(scheduler.jobPlanTable, jobEvent.Job.Name)
		}
	}
}

// 重新计算任务调度状态
func (scheduler *Scheduler) TrySchedule() (scheduleAfer time.Duration) {
	var (
		jobPlan *common.JobSchedulePlan
		now time.Time
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
			// TODO: 尝试执行任务
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

// 调度协程
func (scheduler *Scheduler) scheduleLoop() {
	var (
		jobEvent *common.JobEvent
		scheduleAfter time.Duration
		scheduleTimer *time.Timer
	)
	
	// 初始化计算任务调度状态执行任务
	scheduleAfter = scheduler.TrySchedule()
	
	// 调度的延迟定时器
	scheduleTimer = time.NewTimer(scheduleAfter)
	
	// 定时任务
	for {
		select {
		case jobEvent = <-scheduler.jobEventChan:  // 监听任务变化事件
			// 对内存中的任务列表做增删改查
			scheduler.HandlerJobEvent(jobEvent)
		case <-scheduleTimer.C:  // 最近的任务到期
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
func InitScheduler() (err error){
	G_scheduler = &Scheduler{
		jobEventChan: make(chan *common.JobEvent, 1000),
		jobPlanTable: make(map[string]*common.JobSchedulePlan),
	}
	go G_scheduler.scheduleLoop()
	return
}