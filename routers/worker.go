package routers

import (
	"bytes"
	"dis_control/utils"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

/*
工人要有一个变量宣称自己是否处理连接状态，非连接状态每4s进行一次连接请求，连接成功后3s更新一次心跳
工人自己要有自己的id，
*/

// 声明全局变量
var (
	name         string //手动编辑
	ip           string //手动编辑
	host_address string //手动编辑
	id           string = ""
	cores        int
	useCores     int
	totalCPU     float64
	allCPU       []float64
	isConnected  bool     = false
	isWorking    bool     = false
	caled_signal chan int     //每进行一个单位的计算则向该通道写入一个 1
	caledNums    int      = 0 //记录本次开始工作总共的工作量
	chanStartSig chan int     //每次开始工作时，向该通道写入一个1，防止对isWorking变量的重复读取
)

func InitWorker(r *gin.Engine) {
	wk := r.Group("/worker")
	wk.POST("/gowork", goWork)

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

	//协程：持续请求连接以及发送心跳
	go func() {
		for {
			if !isConnected {
				err := connect_host()
				if err != nil {
					log.Printf("请求连接失败：%v", err)
				}
				time.Sleep(4 * time.Second)
			} else {
				//获取本机CPU信息
				totalCPU, allCPU = utils.Get_CPU()
				//发送heartbeat
				if err := sendHeartBeat(); err != nil {
					log.Println(err)
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

	// go func() {
	// 	for {
	// 		caled_signal <- 1
	// 		time.Sleep(time.Second)
	// 	}
	// }()

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
						if flag {
							//向主机发送消息，自己计算出了目标值
							sendRetMD5(ret)
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
}

type sendLoad struct { //发送数据的类型
	Name  string `json:"name"`
	Ip    string `json:"ip"`
	Cores int    `json:"cores"`
}

// 请求一次连接
func connect_host() error {
	var payLoad struct { //对应接口返回的类型
		Status int    `json:"status"`
		Msg    string `json:"msg"` //对应返回的节点ID
	}

	url := host_address + "/master/getconnect"
	//构建post请求的数据
	sendload := sendLoad{
		Name:  name,
		Ip:    ip,
		Cores: cores,
	}

	// 将数据转换为JSON格式
	jsonData, err := json.Marshal(sendload)
	if err != nil {
		return err
	}
	// 创建HTTP客户端并发送请求
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	//解析响应数据到payLoad结构体中
	err = json.Unmarshal(body, &payLoad)
	if err != nil {
		return err
	}

	if payLoad.Status != 200 {
		return fmt.Errorf("server returned non-200 status: %d %s", payLoad.Status, payLoad.Msg)
	}

	id = payLoad.Msg //赋值本机id
	isConnected = true
	log.Println("连接主机成功")
	return nil
}

type heartStatus struct {
	Id        string    `json:"id"`
	IsWorking bool      `json:"isworking"`
	TotalCPU  float64   `json:"totalcpu"`
	AllCPU    []float64 `json:"allcpu"`
	CaledNums int       `json:"calednums"`
}

// 发送心跳状态：本机id，isWorking，CPU状态信息，已有工作量
func sendHeartBeat() error {
	payLoad := heartStatus{
		Id:        id,
		IsWorking: isWorking,
		TotalCPU:  totalCPU,
		AllCPU:    allCPU,
		CaledNums: caledNums,
	}
	if !isWorking {
		//若为非工作状态则将CPU信息隐藏
		payLoad.TotalCPU = 0
		for i := range payLoad.AllCPU {
			payLoad.AllCPU[i] = 0
		}
		payLoad.CaledNums = 0
	}
	jsondata, err := json.Marshal(payLoad)
	if err != nil {
		return err
	}
	// 创建 HTTP 客户端并发送请求
	url := host_address + "/master/heartbeat"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsondata))
	if err != nil {
		isConnected = false //如果发送请求失败，则可能主机下线，因此取消连接状态，重新进行连接
		return err
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// 解析返回的 JSON 数据
	var responseData struct {
		Status int    `json:"status"`
		Msg    string `json:"msg"`
	}
	if err := json.Unmarshal(body, &responseData); err != nil {
		// fmt.Println("lkjlkajs")
		return err
	}
	if responseData.Status != 200 {
		return fmt.Errorf("resp status not 200: %v", responseData.Msg)
	}
	log.Println("send heartBeat success")
	return nil
}

// 路由函数：由主机唤醒或停止本机的工作状态
// Id, flag ：主机要同时发送该机ID用于验证主机身份
func goWork(c *gin.Context) {
	var payLoad struct {
		Id       string `json:"id"`
		UseCores int    `json:"usecores"`
		Flag     bool   `json:"flag"`
	}
	if err := c.ShouldBindJSON(&payLoad); err != nil {
		c.JSON(400, gin.H{"status": 400, "msg": err.Error()})
	}
	if payLoad.Id != id {
		c.JSON(400, gin.H{"status": 400, "msg": "发送的id与本机id不符"})
	}

	//无错误，准备开始或者停止计算
	useCores = payLoad.UseCores
	var msg string
	if isWorking == payLoad.Flag {
		if isWorking {
			msg = "is still working..."
		} else {
			msg = "is still sleeping..."
		}
	} else {
		isWorking = payLoad.Flag
		if isWorking {
			msg = "success: start work!!!"
			chanStartSig <- 1
		} else {
			msg = "success: stop work!!!"
			caledNums = 0
		}
	}
	log.Println(msg)

	c.JSON(200, gin.H{"status": 200, "msg": msg})
}

type retMD5Load struct {
	Id  string `json:"id"`
	Ret string `json:"ret"`
}

// http函数，向主机发送自己计算获得的目标值
func sendRetMD5(ret string) error {
	myLoad := retMD5Load{
		Id:  id,
		Ret: ret,
	}
	jsonData, err := json.Marshal(myLoad)
	if err != nil {
		return err
	}
	url := host_address + "/master/sendret"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// 解析返回的 JSON 数据
	var responseData struct {
		Status int    `json:"status"`
		Msg    string `json:"msg"`
	}
	if err := json.Unmarshal(body, &responseData); err != nil {
		return err
	}
	if responseData.Status != 200 {
		return fmt.Errorf("resp status not 200: %v", responseData.Msg)
	}
	log.Println(responseData.Msg, "I caled ret:", ret)
	return nil
}
