package master

import (
	"encoding/json"
	"io/ioutil"
)

// 程序配置
type Config struct {
	ApiPort         int `json:"apiPort"`
	ApiReadTimeout  int `json:"apiReadTimeout"`
	ApiWriteTimeout int `json:"apiWriteTimeout"`
}

var (
	G_Config *Config
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
	G_Config = &conf

	return
}
