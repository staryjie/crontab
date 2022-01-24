package common

const (
	// 任务保存目录
	JOB_SAVE_DIR = "/cron/jobs/"

	// 强杀任务目录
	JOB_KILLER_DIR = "/cron/killer/"

	// 强杀任务租约过期时间,单位秒
	KILL_JOB_LEASE_TTL = 1

	// 保存任务事件
	JOB_EVENT_SAVE = 1

	// 任务删除事件
	JOV_EVENT_DELETE = 2

	// 任务杀死事件
	JOB_EVENT_KILL = 3

	// 人物锁目录
	JOB_LOCK_DIR = "/cron/lock/"

	// 查询日志条数
	LOG_LIMIT_NUM = 10

	// 从第几条日志开始查询
	LOG_SKIP_NUM = 0
)
