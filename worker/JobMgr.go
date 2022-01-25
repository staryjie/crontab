package worker

import (
	"context"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/staryjie/crontab/common"
	"time"
)

// 任务管理器
type JobMgr struct {
	client  *clientv3.Client
	kv      clientv3.KV
	lease   clientv3.Lease
	watcher clientv3.Watcher
}

var (
	G_jobMgr *JobMgr
)

// 监听任务变化
func (jobMgr *JobMgr) watchJobs() (err error) {
	var (
		getResp            *clientv3.GetResponse
		kvpair             *mvccpb.KeyValue
		job                *common.Job
		watchStartRevision int64
		watchChan          clientv3.WatchChan
		watchResp          clientv3.WatchResponse
		watchEvent         *clientv3.Event
		jobName            string
		jobEvent           *common.JobEvent
	)
	// 1. get /cron/jobs/ 下所有任务,并且获取当前集群Revision
	if getResp, err = jobMgr.kv.Get(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithPrefix()); err != nil {
		return
	}

	// 遍历任务
	for _, kvpair = range getResp.Kvs {
		// 反序列json 得到job
		if job, err = common.UnpackJob(kvpair.Value); err == nil { // 反解成功
			jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)
			// 把任务同步给调度协程，完成任务调度
			//fmt.Println((*jobEvent).EventType, *(jobEvent).Job)
			G_scheduler.PushJobEvent(jobEvent)
		}
	}

	// 2.从该Revision开始监听事件变化
	go func() {
		// GET时刻的下一个版本
		watchStartRevision = getResp.Header.Revision + 1

		// 启动监听 /cron/jobs/目录的后续变化
		watchChan = jobMgr.watcher.Watch(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithRev(watchStartRevision), clientv3.WithPrefix())

		// 处理监听事件
		for watchResp = range watchChan {
			for _, watchEvent = range watchResp.Events {
				switch watchEvent.Type {
				case mvccpb.PUT: // 任务保存事件
					// 反解json
					if job, err = common.UnpackJob(watchEvent.Kv.Value); err != nil {
						continue // 忽略该次更新事件
					}
					// 构造一个更新事件
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)
					// 推一个更新事件给调度协程
					//fmt.Println((*jobEvent).EventType, *(jobEvent).Job)
					G_scheduler.PushJobEvent(jobEvent)
				case mvccpb.DELETE: // 任务删除事件
					// delete /cron/jobs/job-name
					jobName = common.ExtractJobName(string(watchEvent.Kv.Key))
					job = &common.Job{Name: jobName} // 删除任务只需要jobName即可
					// 构造一个删除事件
					jobEvent = common.BuildJobEvent(common.JOV_EVENT_DELETE, job)
					// 推送一个删除事件给调度协程
					//fmt.Println((*jobEvent).EventType, *(jobEvent).Job)
					G_scheduler.PushJobEvent(jobEvent)
				}
			}
		}
	}()

	return
}

// 监听强杀任务通知
func (jobMgr *JobMgr) watchKiller() {
	var (
		watchChan  clientv3.WatchChan
		watchResp  clientv3.WatchResponse
		watchEvent *clientv3.Event
		jobEvent   *common.JobEvent
		jobName    string
		job        *common.Job
	)
	// 监听/cron/killer/目录
	go func() {
		// 监听/cron/killer/目录的变化
		watchChan = jobMgr.watcher.Watch(context.TODO(), common.JOB_KILLER_DIR, clientv3.WithPrefix())
		// 处理监听事件
		for watchResp = range watchChan {
			for _, watchEvent = range watchResp.Events {
				switch watchEvent.Type {
				case mvccpb.PUT: // 杀死任务
					// 从Etcd的keey中提取任务名
					jobName = common.ExtractKillerName(string(watchEvent.Kv.Key))
					job = &common.Job{Name: jobName}
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_KILL, job)
					// 杀死任务的事件推送给调度器scheduler
					G_scheduler.PushJobEvent(jobEvent)
				case mvccpb.DELETE: // 标记过期，自动被删除
				}
			}
		}
	}()
}

// 初始化管理器
func InitJobMgr() (err error) {
	var (
		config  clientv3.Config
		client  *clientv3.Client
		kv      clientv3.KV
		lease   clientv3.Lease
		watcher clientv3.Watcher
	)

	// 初始化配置
	config = clientv3.Config{
		Endpoints:   G_config.EtcdEndpoints,                                     // Etcd集群
		DialTimeout: time.Duration(G_config.EtcdDialTimeout) * time.Millisecond, // 连接超时时间
	}

	// 建立连接
	if client, err = clientv3.New(config); err != nil {
		return
	}

	// 得到kv和lease API子集
	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)
	watcher = clientv3.NewWatcher(client)

	// 赋值单例
	G_jobMgr = &JobMgr{
		client:  client,
		kv:      kv,
		lease:   lease,
		watcher: watcher,
	}

	// 启动任务监听
	G_jobMgr.watchJobs()

	// 启动监听killer
	G_jobMgr.watchKiller()

	return
}

// 创建任务执行锁
func (jobMgr *JobMgr) CreateJobLock(jobName string) (jobLock *JobLock) {
	// 返回锁
	jobLock = InitJobLock(jobName, jobMgr.kv, jobMgr.lease)

	return
}
