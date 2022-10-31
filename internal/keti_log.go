package internal

import (
	"fmt"
)

const GPU_SCHEDUER_DEBUGG_LEVEL = 1

func KETI_LOG_L1(log string) { //자세한 출력, DumpClusterInfo DumpNodeInfo
	if GPU_SCHEDUER_DEBUGG_LEVEL < 2 { //LEVEL = 1,2,3
		fmt.Println(log)
	}
}

func KETI_LOG_L2(log string) { // 기본출력
	if GPU_SCHEDUER_DEBUGG_LEVEL < 3 { //LEVEL = 2,3
		fmt.Println(log)
	}
}

func KETI_LOG_L3(log string) { //필수출력, 정량용, 에러
	if GPU_SCHEDUER_DEBUGG_LEVEL < 4 { //LEVEL = 3
		fmt.Println(log)
	}
}
