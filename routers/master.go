package routers

import (
	"bytes"
	"dis_control/utils"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var allWorkerNums int = 0 //记录所有已发现的工人数量
var workingNums int = 0   //记录正在工作的工人数量
var connects map[string]nodeStatus
var expiredTime int = 5 //代表多少秒工人未更新心跳则连接过期

type nodeStatus struct {
	id           string //系统赋予的md5码
	name         string //给人看的描述信息
	ip           string
	cores        int
	totalCPU     float64
	allCPU       []float64
	isWorking    bool
	updated_at   time.Time
	startWork_at time.Time //记录开始工作的时间点
	caledNums    int
}

func InitMaster(r *gin.Engine) {
	mst := r.Group("/master")
	mst.POST("/getconnect", getConnect)
	mst.POST("/heartbeat", heartBeat)

	connects = make(map[string]nodeStatus) //键为工人的id，值为其对应的结构体

	go checkHeart()

	go func() {
		for {
			time.Sleep(3 * time.Second)
			log.Println("在线主机数：", allWorkerNums, workingNums)
		}
	}()

	go func() {
		for {
			//模拟控制启动工人节点
			time.Sleep(4 * time.Second)
			for key, _ := range connects {
				if err := goWorkOrNot(key, 1, true); err != nil {
					log.Println(err)
					continue
				}
			}
		}
	}()
}

// 路由函数：接收来自工人节点的连接请求并建立连接
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
	log.Printf("工人 %v 加入{\n id   : %v\n ip   : %v\n cores: %v\n}", newNode.name, newNode.id, newNode.ip, newNode.cores)
	c.JSON(200, gin.H{"status": 200, "msg": id})
}

// 协程：持续检查每个节点的updated_at时间，若超过指定时间则删除其对应的连接
func checkHeart() {
	for {
		for key, value := range connects {
			diff := time.Since(value.updated_at)
			if diff >= time.Duration(expiredTime)*time.Second {
				delete(connects, key)
				log.Printf("工人 %v 下线\n", value.name)
				allWorkerNums--
			}
		}
		time.Sleep(3 * time.Second)
	}
}

// 路由函数：heartBeat，工人需要通过该接口每隔2s向主机发送自己的信息
// 包括id，isWorking，CPU状态信息
func heartBeat(c *gin.Context) {
	var payLoad struct {
		Id        string    `json:"id"`
		IsWorking bool      `json:"isworking"`
		TotalCPU  float64   `json:"totalcpu"`
		AllCPU    []float64 `json:"allcpu"`
		CaledNums int       `json:"calednums"`
	}
	err := c.ShouldBindJSON(&payLoad) //将请求中编码后的json数据解析到payload上
	if err != nil {
		c.JSON(400, gin.H{"status": 400, "msg": err.Error()})
	}

	tempNode, ok := connects[payLoad.Id] // 因为map映射无法直接操作结构体，因此需要用一个temp中转一下
	if ok {
		tempNode.totalCPU = payLoad.TotalCPU
		tempNode.allCPU = payLoad.AllCPU
		tempNode.updated_at = time.Now()
		tempNode.caledNums = payLoad.CaledNums

		connects[payLoad.Id] = tempNode
		c.JSON(200, gin.H{"status": 200, "msg": "success"})
	} else {
		c.JSON(400, gin.H{"status": 400, "msg": "you have expired...建议重启程序"})
	}
}

type sendWorkCmd struct {
	Id       string `json:"id"`
	UseCores int    `json:"usecores"`
	Flag     bool   `json:"flag"`
}

// http: 根据ID向工人发送开始或停止工作指令
func goWorkOrNot(id string, useCores int, flag bool) error {
	sendworkcmd := sendWorkCmd{
		Id:       id,
		UseCores: useCores,
		Flag:     flag,
	}
	jsonData, err := json.Marshal(sendworkcmd)
	if err != nil {
		return err
	}
	url := connects[id].ip + "/worker/gowork"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var respLoad struct {
		Status int    `json:"status"`
		Msg    string `json:"msg"`
	}
	if err = json.Unmarshal(body, &respLoad); err != nil {
		return err
	}
	if respLoad.Status != 200 {
		return fmt.Errorf("status %v . msg: %v", respLoad.Status, respLoad.Msg)
	}

	//指令下发成功
	fmt.Println(respLoad.Msg)
	//修改本地保存的该工人状态
	tempNode := connects[id]
	tempNode.isWorking = flag
	tempNode.startWork_at = time.Now()
	connects[id] = tempNode

	return nil
}
