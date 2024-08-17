package utils

import (
	"fmt"
	"sync"
	"time"

	"math/rand"
)

var length int = 32
var charset = "abcdefghijklmnopqrstuvwxyz0123456789"

func Multi_cal(core int) {
	// rand.Seed(time.Now().UnixNano())
	base_time := 20000000
	fmt.Printf("start %d cores Multi-calculate:\n", core)

	wg := sync.WaitGroup{}
	wg.Add(1)
	for i := 0; i < core; i++ {
		go func(seed int64) {
			defer wg.Done()
			r := rand.New(rand.NewSource(seed))
			nums := 0
			for {
				nums++
				flag := single_cal(r)
				if flag {
					break
				}
				if nums%base_time == 0 {
					fmt.Println(time.Now().Format("2006-01-02 15:05:05"), "*caled..2*10^7")
					nums = 0
				}
			}
		}(time.Now().UnixNano() + int64(i))
	}
	wg.Wait()
}

func single_cal(r *rand.Rand) bool {
	s0 := make([]byte, length)
	for i := 0; i < length; i++ {
		s0[i] = charset[r.Int()%len(charset)]
	}
	s := string(s0)
	fmt.Println(s)
	s1 := Str2md5(s)
	if s == s1 {
		fmt.Println(s, "=相等=", s1)
		return true
	}
	return false
}
