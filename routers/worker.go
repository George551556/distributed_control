package routers

import (
	"dis_control/utils"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
)

/*
工人要有一个变量宣称自己是否处理连接状态，非连接状态每4s进行一次连接请求，连接成功后3s更新一次心跳
工人自己要有自己的id，
*/

// 声明全局变量
var (
	name         string
	ip           string
	host_address string
	host         string
	id           string = ""
	cores        int
	useCores     int
	totalCPU     float64
	allCPU       []float64
	isConnected  bool = false
	conn         *websocket.Conn
	u            url.URL
	isWorking    bool     = false
	caled_signal chan int     //每进行一个单位的计算则向该通道写入一个 1
	caledNums    int      = 0 //记录本次开始工作总共的工作量，停止工作则清零
	chanStartSig chan int     //每次开始工作时，向该通道写入一个1，防止对isWorking变量的重复读取
)

var dialer = websocket.Dialer{
	Proxy: http.ProxyFromEnvironment,
}

// socket通信的消息载体
type WsMessage struct {
	Type        int       `json:"type"`
	Name        string    `json:"name"`
	Cores       int       `json:"cores"`
	TotalCPU    float64   `json:"totalcpu"`
	AllCPU      []float64 `json:"allcpu"`
	IsWorking   bool      `json:"isworking"`
	UseCores    int       `json:"usecores"`
	StartWorkAt time.Time `json:"startwork_at"`
	CaledNums   int       `json:"calednums"`
	Result      string    `json:"result"`
}

func InitWorker() {
	//全局变量赋初值
	cores = 4
	useCores = 0
	totalCPU = 51.1
	allCPU = []float64{1, 2, 3, 4}
	caled_signal = make(chan int, 10)
	chanStartSig = make(chan int, 2)
	name = viper.GetString("name")
	ip = fmt.Sprintf("http://%v:%v", viper.GetString("local_address"), viper.GetInt("local_port"))
	host_address = fmt.Sprintf("http://%v:%v", viper.GetString("host_address"), viper.GetInt("host_port"))
	host = fmt.Sprintf("%v:%v", viper.GetString("host_address"), viper.GetInt("host_port"))
	u = url.URL{Scheme: "ws", Host: host, Path: "/master/myws"}
	cores = viper.GetInt("cores")

	//协程：持续请求连接以及发送心跳
	go func() {
		for {
			if !isConnected {
				connect_host()
				time.Sleep(4 * time.Second)
			} else {
				//获取本机CPU信息，发送heartbeat
				totalCPU, allCPU = utils.Get_CPU()
				if !sendHeartBeat() {
					conn.Close()
					isConnected = false
				}
				time.Sleep(time.Second)
			}
		}
	}()

	//读通道
	go func() {
		for x := range caled_signal {
			caledNums += x
		}
	}()

	//进行md5计算的主要协程，由管道来控制主死循环的执行与停止
	go func() {
		for startSignal := range chanStartSig {
			startSignal += 1
			base_time := 20000000
			fmt.Printf("start %d cores Multi-calculate:\n", useCores)

			for i := 0; i < useCores; i++ {
				go func(seed int64) {
					r := rand.New(rand.NewSource(seed))
					nums := 0
					for {
						nums++
						ret, flag := utils.Single_cal(r)
						if nums == 20000000-1000000 {
							flag = true
							ret = "hahah just kidding!!!"
							fmt.Printf("ret: %v\n", ret)
						}
						if flag {
							//向主机发送消息，自己计算出了目标值
							sendRetMD5(ret)
							isWorking = false
							break
						}
						if !isWorking {
							break
						}
						if nums%base_time == 0 {
							// log.Println("*caled..2*10^7")
							caled_signal <- 1
							log.Println("one unit calculate...")
							nums = 0
						}
					}
				}(time.Now().UnixNano() + int64(i))
			}
		}
	}()

	//从socket中循环读取消息
	for {
		if isConnected {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("读取消息失败: ", err)
				conn.Close()
				isConnected = false
			}
			// fmt.Println(string(message))
			//处理收到的消息
			var msg WsMessage
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Println("消息解码错误: ", err)
			}
			if msg.Type != 3 {
				log.Println("消息类型码不合法...")
			}
			goWork(msg.IsWorking, msg.UseCores)
		} else {
			time.Sleep(3 * time.Second)
		}
	}
}

// 请求一次连接
func connect_host() bool {
	var err error
	conn, _, err = dialer.Dial(u.String(), nil)
	if err != nil {
		log.Println("ws连接失败：", err)
		return false
	}
	log.Println("连接成功！！！")
	isConnected = true
	return true
}

// 发送心跳状态：名称，isWorking，CPU状态信息，已有工作量
func sendHeartBeat() bool {
	msg := WsMessage{
		Type:      1,
		Name:      name,
		Cores:     cores,
		TotalCPU:  totalCPU,
		AllCPU:    allCPU,
		IsWorking: isWorking,
		CaledNums: caledNums,
	}
	if !isWorking {
		//若为非工作状态则将CPU信息隐藏
		msg.TotalCPU = 0
		for i := range msg.AllCPU {
			msg.AllCPU[i] = 0
		}
		msg.CaledNums = 0
	}
	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Println("消息序列化失败: ", err)
		return false
	}
	// 发送消息
	if err := conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
		log.Println("发送失败：", err)
		return false
	}
	return true
}

// 由主机唤醒或停止本机的工作状态
// 参数：开始或停止工作的指令，使用核数
func goWork(shouldWork bool, shouldUseCores int) {
	//准备开始或者停止计算
	useCores = shouldUseCores
	var msg string
	if isWorking == shouldWork {
		if isWorking {
			msg = "is still working..."
		} else {
			msg = "is still sleeping..."
		}
	} else {
		isWorking = shouldWork
		if isWorking {
			msg = "success: start work!!!"
			chanStartSig <- 1
		} else {
			msg = "success: stop work!!!"
			caledNums = 0
		}
	}
	log.Println(msg)

}

// 向主机发送自己计算获得的目标值
func sendRetMD5(ret string) bool {
	msg := WsMessage{
		Type:   2,
		Result: ret,
	}
	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Println("消息编码失败：", err)
		return false
	}
	if err := conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
		log.Println("发送失败：", err)
		return false
	}
	return true
}
