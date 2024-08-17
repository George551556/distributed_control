package utils

import (
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
)

func Get_CPU() (float64, []float64) {
	totalCPU, _ := cpu.Percent(time.Second, false)
	allCPU, _ := cpu.Percent(time.Second, true)
	// totalCPU = fmt.Sprintf("%.4f", totalCPU)
	// for i := range allCPU {
	// 	allCPU[i] = fmt.Sprintf("%.4f", allCPU[i])
	// }
	return totalCPU[0], allCPU
}
