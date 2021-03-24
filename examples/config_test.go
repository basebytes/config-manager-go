package examples

import (
	"fmt"
	"testing"
	"time"

	"github.com/runnerhbc/config-manager-go/config"
	"github.com/spf13/viper"

)

func TestCfgManager(t *testing.T) {
	var conf = MyConfig{
		Item: "cfgItem",
	}
	cfgManager := config.New(&ViperLoader{}, "config.toml",
		config.WithDefaultConfigs(map[string]interface{}{"mysql": map[string]interface{}{"port": 3306}}))
	if err := cfgManager.ReadConfig(&conf); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(conf)
	}
	for {
		time.Sleep(time.Second * 10)
		fmt.Println(conf)
	}
}

type MyConfig struct {
	Mysql struct {
		DBName   string
		Host     string
		UserName string
		Pass     string
		Port     uint32
	}
	Item string
}

type ViperLoader struct {
}

func (vl *ViperLoader) Load(path string, v interface{}) error {
	viper.SetConfigFile(path)
	err := viper.ReadInConfig()
	if err == nil {
		err = viper.Unmarshal(&v)
	}
	return err
}

func (vl *ViperLoader) Save(path string, v interface{}) error {
	viper.SetConfigFile(path)
	return viper.WriteConfig()
}
