package utils

import (
	"fmt"
	"log"
	"sync"
	"time"

	"math/rand"
)

var length int = 32
var charset = "abcdef0123456789"
var lth int = 16

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
				ret, flag := Single_cal(r)
				if flag {
					fmt.Println(ret)
					break
				}
				if nums%base_time == 0 {
					log.Println("*caled..2*10^7")
					nums = 0
				}
			}
		}(time.Now().UnixNano() + int64(i))
	}
	wg.Wait()
}

func Single_cal(r *rand.Rand) (string, bool) {
	s0 := make([]byte, length)
	for i := 0; i < length; i++ {
		s0[i] = charset[r.Int()%lth]
	}
	s := string(s0)
	s1 := Str2md5(s)
	if s == s1 {
		return s, true
	}
	return "", false
}
