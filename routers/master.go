package routers

import (
	"dis_control/utils"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

var allWorkerNums int = 0 //记录所有已发现的工人数量
var workingNums int = 0   //记录正在工作的工人数量
var connects map[string]nodeStatus
var expiredTime int = 10 //代表多少秒工人未更新心跳则连接过期

type nodeStatus struct {
	id         string //系统赋予的md5码
	name       string //给人看的描述信息
	ip         string
	cores      int
	totalCPU   float64
	allCPU     []float64
	isWorking  bool
	updated_at time.Time
}

func InitMaster(r *gin.Engine) {
	mst := r.Group("/master")
	mst.POST("/getconnect", getConnect)
	mst.POST("/heartbeat", heartBeat)

	connects = make(map[string]nodeStatus) //键为工人的id，值为其对应的结构体
	go checkHeart()
	go func() {
		for {
			log.Println(allWorkerNums, workingNums)
			time.Sleep(2 * time.Second)
		}
	}()
}

// 接收来自工人节点的连接请求并建立连接
// 要求发送的信息包括：name, ip, cores
func getConnect(c *gin.Context) {
	var payLoad struct {
		Name  string `json:"name"`
		Ip    string `json:"ip"`
		Cores int    `json:"cores"`
	}
	if err := c.ShouldBindJSON(&payLoad); err != nil {
		log.Println(err)
		c.JSON(500, gin.H{"status": 500, "msg": "wrong cores content"})
	}

	id := utils.GetRandom_md5()
	newNode := nodeStatus{
		id:         id,
		name:       payLoad.Name,
		ip:         payLoad.Ip,
		cores:      payLoad.Cores,
		totalCPU:   0.00,
		allCPU:     make([]float64, payLoad.Cores),
		isWorking:  false,
		updated_at: time.Now(),
	}
	connects[id] = newNode
	allWorkerNums++
	log.Printf("工人 %v 加入{\n id:%v\n ip:%v\n cores:%v\n}", newNode.name, newNode.id, newNode.ip, newNode.cores)
	c.JSON(200, gin.H{"status": 200, "msg": id})
}

// 协程：持续检查每个节点的updated_at时间，若超过指定时间则删除其对应的连接
func checkHeart() {
	for {
		for key, value := range connects {
			diff := time.Since(value.updated_at)
			if diff >= time.Duration(expiredTime)*time.Second {
				delete(connects, key)
				allWorkerNums--
			}
		}
		time.Sleep(3 * time.Second)
	}
}

// 路由函数：heartBeat，工人需要通过该接口每隔2s向主机发送自己的信息
// 包括id，isWorking，CPU状态信息
func heartBeat(c *gin.Context) {
	fmt.Println("heartbeat start")
	var payLoad struct {
		Id        string    `json:"id"`
		IsWorking bool      `json:"isworking"`
		TotalCPU  float64   `json:"totalcpu"`
		AllCPU    []float64 `json:"allcpu"`
	}
	err := c.ShouldBindJSON(&payLoad) //将请求中编码后的json数据解析到payload上
	if err != nil {
		c.JSON(400, gin.H{"status": 400, "msg": err.Error()})
	}

	tempNode, ok := connects[payLoad.Id] // 因为map映射无法直接操作结构体，因此需要用一个temp中转一下
	if ok {
		if tempNode.isWorking {
			tempNode.totalCPU = payLoad.TotalCPU
			tempNode.allCPU = payLoad.AllCPU
			tempNode.updated_at = time.Now()
			connects[payLoad.Id] = tempNode
		} else {
			tempNode.updated_at = time.Now()
			connects[payLoad.Id] = tempNode
		}
		c.JSON(200, gin.H{"status": 200, "msg": "success"})
	} else {
		c.JSON(400, gin.H{"status": 400, "msg": "you have expired...建议重启程序"})
	}
}
