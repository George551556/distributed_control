package main

import (
	"dis_control/routers"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	var nodeType int
	var err error
	if len(os.Args) == 2 {
		nodeType, err = strconv.Atoi(os.Args[1])
		if err != nil {
			log.Fatal("不合法的命令行参数")
		}
	} else {
		nodeType = 1
	}
	r := gin.Default()
	//加载模板目录下模板文件
	r.LoadHTMLGlob("templates/*")

	//路由初始化
	r.GET("/lt", littleTest)
	if nodeType == 0 {
		log.Println("以主节点身份启动")
		routers.InitMaster(r)
		r.Run(":8000")
	} else {
		log.Println("默认以工人节点身份启动")
		routers.InitWorker(r)
		r.Run(":8001")
	}

	// r.Run(":8000")
}

func littleTest(c *gin.Context) {
	c.HTML(200, "index.html", gin.H{"msg": "123"})
}
