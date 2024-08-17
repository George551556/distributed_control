package test

import (
	"fmt"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
)

func TestAa(t *testing.T) {
	//获取cpu每个核的占用信息
	for i := 0; i < 3; i++ {
		totalCPU, _ := cpu.Percent(time.Second, false)
		allCPU, _ := cpu.Percent(time.Second, true)
		fmt.Println(totalCPU, allCPU)
		time.Sleep(time.Second)
	}
	// totalCPU = fmt.Sprintf("%.4f", totalCPU)
	// for i := range allCPU {
	// 	allCPU[i] = fmt.Sprintf("%.4f", allCPU[i])
	// }

	// 测试md5的功能
	// fmt.Println(utils.GetRandom_md5())

	// for i := 65; i < 123; i++ {
	// 	fmt.Println(string(byte(i)))
	// }
}
