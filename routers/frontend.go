package routers

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

/*
这里放置返回前端页面的路由函数以及与web用户交互的路由逻辑
*/
func InitFront(r *gin.Engine) {
	front := r.Group("/front")
	front.GET("/gowork", index)
	front.GET("/getmaindata", returnData)
	front.POST("/goworkornot", frt_gowork)
	front.POST("/batchctrl", batchControl)

}

// 返回主页面
func index(c *gin.Context) {
	c.HTML(200, "gowork.html", gin.H{})
}

// 路由函数：返回主要数据
func returnData(c *gin.Context) {
	time := fmt.Sprintf("刷新时间：%v", time.Now().Format("2006-01-02 15:04:05"))
	workerNum, finalSuccess, result, data := GetMainData()
	c.JSON(200, gin.H{
		"date-time":    time,
		"worker":       workerNum,
		"finalsuccess": finalSuccess,
		"result":       result,
		"data":         data,
	})
}

// 路由函数：接受用户的工作或停止请求
func frt_gowork(c *gin.Context) {
	id := c.PostForm("id")
	temp_usecores := c.PostForm("usecores")
	temp_isworking := c.PostForm("isworking")
	var usecores int
	var isworking bool
	usecores, err := strconv.Atoi(temp_usecores)
	if err != nil {
		log.Println(err)
		c.JSON(400, gin.H{"msg": "wrong usecores type"})
	}
	if temp_isworking == "true" {
		isworking = true
	} else if temp_isworking == "false" {
		isworking = false
	} else {
		log.Println("wrong isworking type")
		c.JSON(400, gin.H{"msg": "wrong isworking type"})
	}

	//向master请求
	if ok := GoWorkOrNot(id, usecores, isworking); !ok {
		log.Println("3452error:", err)
		c.JSON(400, gin.H{"msg": "3452error"})
	}

	c.JSON(200, gin.H{})
}

// 路由函数：批量启动或停止所有的工作节点
func batchControl(c *gin.Context) {
	temp_slt := c.PostForm("slt")
	slt, err := strconv.Atoi(temp_slt)
	if err != nil {
		log.Println(err)
		c.JSON(400, gin.H{"msg": "wrong usecores type"})
	}
	if err := Mst_batchCtrl(slt); err != nil {
		c.JSON(400, gin.H{"msg": err.Error()})
	}
	c.JSON(200, gin.H{})
}
