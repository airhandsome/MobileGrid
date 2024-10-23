package util

import (
	"fmt"
	"github.com/spf13/viper"
)

func InitConfig() {
	viper.SetConfigName("default") // 配置文件名（不带扩展名）
	viper.SetConfigType("yaml")    // 如果配置文件没有扩展名，则需要指定类型
	viper.AddConfigPath("config")  // 指定查找配置文件的路径
	err := viper.ReadInConfig()    // 查找并读取配置文件
	if err != nil {                // 处理读取配置文件的错误
		panic(fmt.Errorf("Fatal error config file: %w \n", err))
	}
}
