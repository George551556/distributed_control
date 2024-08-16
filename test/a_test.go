package test

import (
	"dis_control/utils"
	"fmt"
	"testing"
)

func TestAa(t *testing.T) {
	//获取cpu每个核的占用信息
	// for i := 0; i < 1; i++ {
	// 	percent, _ := cpu.Percent(time.Second, true)
	// 	fmt.Printf("percent: %5v\n", percent)
	// 	time.Sleep(time.Second)
	// }

	// 测试md5的功能
	fmt.Println(utils.GetRandom_md5())

	// for i := 65; i < 123; i++ {
	// 	fmt.Println(string(byte(i)))
	// }
}
