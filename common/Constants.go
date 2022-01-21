package common

const (
	// 任务保存目录
	JOB_SAVE_DIR = "/cron/jobs/"

	// 强杀任务目录
	JOB_KILLER_DIR = "/cron/killer/"

	// 强杀任务租约过期时间,单位秒
	KILL_JOB_LEASE_TTL = 1
)
