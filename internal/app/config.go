package app

import (
	"encoding/json"
	"os"
)

// 配置结构体
type Config struct {
	Title          string // 标题
	Port           string // 服务器开放端口
	AdminPasswdMD5 string // 管理员密码MD5
}

// 检测配置文件是否存在，不存在则生成
func checkConfig() error {
	exi, err := exists("config.json")
	if err != nil {
		return err
	}
	if !exi {
		f, err := os.Create("config.json")
		if err != nil {
			return err
		}
		defer f.Close()

		// 写入默认配置
		var defaultCfg Config
		defaultCfg.Title = "澄心OJ - chengxinoj"
		defaultCfg.Port = ":8080"
		defaultCfg.AdminPasswdMD5 = ""
		bts, err := json.MarshalIndent(defaultCfg, "", "    ")
		if err != nil {
			return err
		}
		_, err = f.Write(bts)
		if err != nil {
			return err
		}
	}
	return nil
}

// 加载配置文件
func readConfigTo(c *Config) error {
	b, err := os.ReadFile("config.json")
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, c)
	if err != nil {
		return err
	}
	return nil
}
