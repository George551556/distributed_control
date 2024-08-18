package main

import (
	"dis_control/routers"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func main() {
	// rand.Seed(time.Now().UnixNano())
	var nodeType int //接收命令行参数确定节点启动类型
	var err error
	if len(os.Args) == 2 {
		nodeType, err = strconv.Atoi(os.Args[1])
		if err != nil {
			log.Fatal("不合法的命令行参数")
		}
	} else {
		nodeType = 1
	}
	//viper读取配置文件config.json
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath("./")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal(err)
	}
	host_port := viper.GetInt("host_port")
	local_port := viper.GetInt("local_port")

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	//路由初始化
	r.GET("/", littleTest)
	if nodeType == 0 {
		log.Println("以主节点身份启动")
		routers.InitMaster(r)
		routers.InitFront(r)
		r.Run(fmt.Sprintf(":%v", host_port))
	} else {
		log.Println("默认以工人节点身份启动")
		routers.InitWorker(r)
		r.Run(fmt.Sprintf(":%v", local_port))
	}

	// r.Run(":8000")
}

func littleTest(c *gin.Context) {
	c.HTML(200, "index.html", gin.H{"msg": "123"})
}
