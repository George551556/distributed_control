package routers

import (
	"bytes"
	"dis_control/utils"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var allWorkerNums int = 0                      //记录所有已发现的工人数量
var workingNums int = 0                        //记录正在工作的工人数量
var connects = make(map[string]nodeStatus)     //键为工人的id，值为其对应的结构体信息
var wsConns = make(map[string]*websocket.Conn) //键为工人的id，值为其对应的webSocket连接对象
var expiredTime int = 8                        //代表多少秒工人未更新心跳则连接过期
var finalSuccess bool = false
var result []string

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域请求
	},
}

type nodeStatus struct {
	ID          string    `json:"id"` // 使用json标签来指定序列化后的字段名
	Name        string    `json:"name"`
	IP          string    `json:"ip"`
	Cores       int       `json:"cores"`
	TotalCPU    float64   `json:"totalCPU"`
	AllCPU      []float64 `json:"allCPU"`
	IsWorking   bool      `json:"isWorking"`
	UpdatedAt   time.Time `json:"updated_at"` // 时间字段通常可以自动序列化为ISO 8601格式
	StartWorkAt time.Time `json:"startWork_at"`
	CaledNums   int       `json:"caledNums"`
}

func InitMaster(r *gin.Engine) {
	mst := r.Group("/master")
	mst.POST("/getconnect", getConnect)
	mst.POST("/heartbeat", heartBeat)
	mst.POST("/sendret", sendRet)

	// connects = make(map[string]nodeStatus)

	go checkHeart()
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
		ID:        id,
		Name:      payLoad.Name,
		IP:        payLoad.Ip,
		Cores:     payLoad.Cores,
		TotalCPU:  0.00,
		AllCPU:    make([]float64, payLoad.Cores),
		IsWorking: false,
		UpdatedAt: time.Now(),
	}
	connects[id] = newNode
	allWorkerNums++
	log.Printf("工人 %v 加入{\n id   : %v\n ip   : %v\n cores: %v\n}", newNode.Name, newNode.ID, newNode.IP, newNode.Cores)
	c.JSON(200, gin.H{"status": 200, "msg": id})
}

// 协程：持续检查每个节点的updated_at时间，若超过指定时间则删除其对应的连接。并更新工作节点数
func checkHeart() {
	for {
		tempWorking := 0
		for key, value := range connects {
			diff := time.Since(value.UpdatedAt)
			if diff >= time.Duration(expiredTime)*time.Second {
				delete(connects, key)
				log.Printf("工人 %v 下线\n", value.Name)
				allWorkerNums--
			}
			if value.IsWorking {
				tempWorking++
			}
		}
		workingNums = tempWorking

		if finalSuccess {
			log.Printf("!!!! This is result [%v] !!!:", result)
		}
		time.Sleep(2 * time.Second)
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
		tempNode.TotalCPU = payLoad.TotalCPU
		tempNode.AllCPU = payLoad.AllCPU
		tempNode.UpdatedAt = time.Now()
		tempNode.CaledNums = payLoad.CaledNums

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
func GoWorkOrNot(id string, useCores int, flag bool) error {
	sendworkcmd := sendWorkCmd{
		Id:       id,
		UseCores: useCores,
		Flag:     flag,
	}
	jsonData, err := json.Marshal(sendworkcmd)
	if err != nil {
		return err
	}
	url := connects[id].IP + "/worker/gowork"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
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
	tempNode.IsWorking = flag
	tempNode.StartWorkAt = time.Now()
	connects[id] = tempNode

	return nil
}

// 路由函数：接收工人发送的md5目标值，同时停止所有工人的工作
func sendRet(c *gin.Context) {
	var myLoad struct {
		Id  string `json:"id"`
		Ret string `json:"ret"`
	}
	if err := c.ShouldBindJSON(&myLoad); err != nil {
		c.JSON(400, gin.H{"status": 400, "msg": err.Error()})
	}
	tempNode, ok := connects[myLoad.Id]
	//停止所有工人的计算工作
	finalSuccess = true
	var item_ret string
	if ok {
		item_ret = tempNode.Name + " !!!!!!!!!!!caled the result: " + myLoad.Ret
	} else {
		item_ret = "未知用户 " + " !!!!!!!!!!!caled the result: " + myLoad.Ret
	}
	result = append(result, item_ret)
	for key, value := range connects {
		if err := GoWorkOrNot(key, 0, false); err != nil {
			log.Println(value.Name, "stop work ERROR:", err)
		}
	}
	c.JSON(200, gin.H{"status": 200, "msg": "Congratulations !!!"})
}

// 辅助函数：向前端接口返回切片形式的已连接节点信息
func GetMainData() (int, int, bool, []string, []nodeStatus) {
	var mySlc []nodeStatus
	for _, value := range connects {
		mySlc = append(mySlc, value)
	}
	//对切片按isWorking排序
	sort.Slice(mySlc, func(i int, j int) bool {
		return mySlc[i].IsWorking && !mySlc[j].IsWorking
	})

	return allWorkerNums, workingNums, finalSuccess, result, mySlc
}

// 辅助函数：根据参数slt，批量操作所有工作节点
func Mst_batchCtrl(slt int) error {
	if slt == 0 {
		for key := range connects {
			if err := GoWorkOrNot(key, 0, false); err != nil {
				return err
			}
		}
	} else if slt == 1 {
		for key := range connects {
			if err := GoWorkOrNot(key, 1, true); err != nil {
				return err
			}
		}
	} else if slt == 2 {
		for key, value := range connects {
			fullCore := value.Cores
			if err := GoWorkOrNot(key, fullCore, true); err != nil {
				return err
			}
		}
	} else {
		return fmt.Errorf("slt参数数值不合法")
	}
	return nil
}
