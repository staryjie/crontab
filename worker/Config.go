package worker

import (
	"encoding/json"
	"io/ioutil"
)

// 程序配置
type Config struct {
	EtcdEndpoints   []string `json:"etcdEndpoints"`
	EtcdDialTimeout int      `json:"etcdDialTimeout"`
	MongodbUri string `json:"mongodbUri"`
	MongodbCollectTimeout int `json:"mongodbCollectTimeout"`
	JobLogStoreDb string `json:"jobLogStoreDb"`
	JobLogStoreCollection string `json:"jobLogStoreCollection"`
	JobLogBatchSize int `json:"jobLogBatchSize"`
	JobLogCommitTimeout int `json:"jobLogCommitTimeout"`
}

var (
	G_config *Config
)

// 加载配置
func InitConfig(configFile string) (err error) {
	var (
		content []byte
		conf    Config
	)
	// 1.读配置文件
	if content, err = ioutil.ReadFile(configFile); err != nil {
		return
	}

	// 2.json反序列化
	if err = json.Unmarshal(content, &conf); err != nil {
		return
	}

	// 3.赋值单例
	G_config = &conf

	return
}
