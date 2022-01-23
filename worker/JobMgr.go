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
func (JobMgr *JobMgr) watchJobs() (err error) {
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
	if getResp, err = G_jobMgr.kv.Get(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithPrefix()); err != nil {
		return
	}

	// 遍历任务
	for _, kvpair = range getResp.Kvs {
		// 反序列json 得到job
		if job, err = common.UnpackJob(kvpair.Value); err == nil { // 反解成功
			jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)
			// TODO: 把任务同步给调度协程，完成任务调度
		}
	}

	// 2.从该Revision开始监听事件变化
	go func() {
		// GET时刻的下一个版本
		watchStartRevision = getResp.Header.Revision + 1

		// 启动监听 /cron/jobs/目录的后续变化
		watchChan = G_jobMgr.watcher.Watch(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithRev(watchStartRevision))

		// 处理监听事件
		for watchResp = range watchChan {
			for watchEvent = range watchResp.Events {
				switch watchEvent.Type {
				case mvccpb.PUT: // 任务保存事件
					// 反解json
					if job, err = common.UnpackJob(watchEvent.Kv.Value); err != nil {
						continue // 忽略该次更新事件
					}
					// 构造一个更新事件
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)
					// TODO: 反序列化job，推一个更新事件给调度协程
				case mvccpb.DELETE: // 任务删除事件
					// delete /cron/jobs/job-name
					jobName = common.ExtractJobName(string(watchEvent.Kv.Key))
					job = &common.Job{Name: jobName} // 删除任务只需要jobName即可
					// 构造一个删除事件
					jobEvent = common.BuildJobEvent(common.JOV_EVENT_DELETE, job)
					// TODO: 推送一个删除事件给调度协程
				}
			}
		}
	}()

	return
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
		Endpoints:   G_Config.EtcdEndpoints,                                     // Etcd集群
		DialTimeout: time.Duration(G_Config.EtcdDialTimeout) * time.Millisecond, // 连接超时时间
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

	return
}
