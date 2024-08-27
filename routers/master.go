package routers

import (
	"dis_control/utils"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var allWorkerNums int = 0                      //记录所有已发现的工人数量
var connects = make(map[string]nodeStatus)     //键为工人的id，值为其对应的结构体信息
var wsConns = make(map[string]*websocket.Conn) //键为工人的id，值为其对应的webSocket连接对象
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
	Cores       int       `json:"cores"`
	TotalCPU    float64   `json:"totalCPU"`
	AllCPU      []float64 `json:"allCPU"`
	IsWorking   bool      `json:"isWorking"`
	UpdatedAt   time.Time `json:"updated_at"` // 时间字段通常可以自动序列化为ISO 8601格式
	StartWorkAt string    `json:"startWork_at"`
	UseCores    int       `json:"usecores"`
	CaledNums   int       `json:"caledNums"`
}

func InitMaster(r *gin.Engine) {
	mst := r.Group("/master")
	mst.GET("/myws", myWS)

	// connects = make(map[string]nodeStatus)

}

// 处理来自工人节点的连接请求
// 要求发送的信息包括：name, ip, cores
func myWS(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
	}
	// defer ws.Close()
	//保存连接对象
	id := utils.GetRandom_md5()
	wsConns[id] = ws
	//开协程持续接收消息
	go func() {
		for {
			tempNode, ok := workerMsgExist(id)
			//根据id查看
			_, message, err := ws.ReadMessage()
			if err != nil {
				log.Println("读取消息失败: ", err)
				memberOut(id)
				break
			}
			var msg WsMessage
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Println("解码消息失败")
				memberOut(id)
				break
			}
			if msg.Type == 1 {
				if ok {
					//说明工人信息已存在，更新心跳信息即可
					tempNode.TotalCPU = msg.TotalCPU
					tempNode.AllCPU = msg.AllCPU
					tempNode.CaledNums = msg.CaledNums
					tempNode.IsWorking = msg.IsWorking
					tempNode.StartWorkAt = msg.StartWorkAt
					tempNode.UseCores = msg.UseCores
					connects[id] = tempNode
				} else {
					//向connects中添加一个新的对象
					newNode := nodeStatus{
						ID:    id,
						Name:  msg.Name,
						Cores: msg.Cores,
					}
					connects[id] = newNode
					log.Printf("工人 %v 上线\n", msg.Name)
					allWorkerNums++
				}
			} else if msg.Type == 2 {
				item_ret := fmt.Sprintf("%v计算出了结果：%v", connects[id].Name, msg.Result)
				result = append(result, item_ret)
				finalSuccess = true
				Mst_batchCtrl(0)
			} else {
				log.Println("消息类型码不合法...")
			}
		}
	}()
}

// 辅助函数：根据id查看工人信息结构体中是否已经存在
func workerMsgExist(id string) (nodeStatus, bool) {
	tempNode, ok := connects[id]
	return tempNode, ok
}

// 辅助函数：根据id删除两个map中的信息，并更新相关全局变量
func memberOut(id string) {
	log.Printf("工人 %v 下线...", connects[id].Name)
	delete(connects, id)
	delete(wsConns, id)
	allWorkerNums--
}

// 根据ID向工人发送开始或停止工作指令
func GoWorkOrNot(id string, useCores int, flag bool) bool {
	msg := WsMessage{
		Type:      3,
		IsWorking: flag,
		UseCores:  useCores,
	}
	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Println("workOrNot 消息编码失败: ", err)
		return false
	}
	if err := wsConns[id].WriteMessage(websocket.TextMessage, jsonData); err != nil {
		log.Println("workOrNot 消息发送失败: ", err)
		return false
	}
	return true
}

// 辅助函数：向前端接口返回切片形式的已连接节点信息
func GetMainData() (int, bool, []string, []nodeStatus) {
	var mySlc []nodeStatus
	for _, value := range connects {
		mySlc = append(mySlc, value)
	}
	//对切片按isWorking排序
	sort.Slice(mySlc, func(i int, j int) bool {
		if mySlc[i].IsWorking != mySlc[j].IsWorking {
			return mySlc[i].IsWorking && !mySlc[j].IsWorking
		} else {
			return mySlc[i].CaledNums > mySlc[j].CaledNums
		}
	})

	return allWorkerNums, finalSuccess, result, mySlc
}

// 辅助函数：根据参数slt，批量操作所有工作节点
// 0：停止所有节点工作    1：全部单核运行      2：全部满载运行
func Mst_batchCtrl(slt int) error {
	if slt == 0 {
		for key := range connects {
			log.Println("停止所有节点工作")
			GoWorkOrNot(key, 0, false)
		}
	} else if slt == 1 {
		log.Println("全部单核运行")
		for key := range connects {
			GoWorkOrNot(key, 1, true)
		}
	} else if slt == 2 {
		log.Println("全部满载运行")
		for key, value := range connects {
			fullCore := value.Cores
			GoWorkOrNot(key, fullCore, true)
		}
	} else {
		return fmt.Errorf("slt参数数值不合法")
	}
	return nil
}

// 辅助函数：根据传入的id向工人节点发送消息，清零其工作量
func Mst_calNumClear(id string) bool {
	tempNode, ok := workerMsgExist(id)
	if !ok {
		log.Println("Mst_calNumClear error: worker disconnected...")
		return true
	}
	if tempNode.CaledNums == 0 {
		return true
	}
	msg := WsMessage{
		Type: 4,
	}
	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Println("Mst_calNumClear 消息编码失败: ", err)
		return false
	}
	if err := wsConns[id].WriteMessage(websocket.TextMessage, jsonData); err != nil {
		log.Println("Mst_calNumClear 消息发送失败: ", err)
		return false
	}
	return true
}
