package main

import (
	database "FSMTestingPlatform/Database"
	"FSMTestingPlatform/Utils"
	"context"
	"flag"
	"fmt"
	"time"
)

func main()  {

	action := flag.String("action", "", "the action of starting redis")
	flag.Parse()

	if *action != "" {
		switch *action {
		case "vci_fault_model3":
			vciFault2Redis3()
		case "vci_fault_model5":
			vciFault2Redis5()
		case "pmm_fault_model3":
			pmmFault2Redis3()
		case "pmm_fault_model5":
			pmmFault2Redis5()
		case "pmm_alarm_model3":
			pmmAlarm2Redis3()
		case "pmm_alarm_model5":
			pmmAlarm2Redis5()
		case "ohp_alarm_model3":
			ohpAlarm2Redis3()
		case "ohp_alarm_model5":
			ohpAlarm2Redis5()
		default:
			fmt.Printf("unknow instruction\n")
		}
	}

}

// vciFault2Redis
// vci故障码存入Redis
func vciFault2Redis3() {
	ctx := context.Background()
	database.Redis.Del(ctx, "vci_fault_model3")
	faultList := []int{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d,
		0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c,
		0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f,
		0x30, 0x31, 0x32, 0x33,
		0x40, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48, 0x49, 0x4a, 0x4b, 0x4c, 0x4d, 0x4e, 0x4f,
		0x50, 0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57, 0x58, 0x59, 0x5a, 0x5b, 0x5c, 0x5d, 0x5e, 0x5f,
		0x60,
	}
	strModel := Utils.IntArr2StringArr(faultList)
	Utils.ArrayLow2Redis(ctx, "vci_fault_model3", strModel, []int{0, 1, 2}, len(strModel))
}

func vciFault2Redis5() {
	timeEnd := false
	ctx := context.Background()
	database.Redis.Del(ctx, "vci_fault_model5")
	faultList := []int{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d,
		0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c,
		0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f,
		0x30, 0x31, 0x32, 0x33,
		0x40, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48, 0x49, 0x4a, 0x4b, 0x4c, 0x4d, 0x4e, 0x4f,
		0x50, 0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57, 0x58, 0x59, 0x5a, 0x5b, 0x5c, 0x5d, 0x5e, 0x5f,
		0x60,
	}
	strModel := Utils.IntArr2StringArr(faultList)

	var timer = time.NewTicker(5*time.Millisecond)
	go func() {
		Utils.ArrayHig2Redis(ctx, "vci_fault_model5", strModel, []int{0, 1, 2, 3, 4}, len(strModel))
		// fmt.Println("defer", time.Now().Format("2006-01-02 15:04:05"))
		timeEnd = true
	}()

	for  {
		select {
		case <-timer.C:
			if timeEnd {
				goto endTime
			}
			timer = time.NewTicker(5*time.Millisecond)
			database.RedisPipe.Exec(ctx)
		}
	}
endTime:
	fmt.Printf("time end\n")
}

// pmmFault2Redis
// PMM FaultType 故障类型枚举 Redis
func pmmFault2Redis3() {
	ctx := context.Background()
	database.Redis.Del(ctx, "pmm_fault_model3")
	faultList := []int{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C,
	}
	strModel3 := Utils.IntArr2StringArr(faultList)
	Utils.ArrayLow2Redis(ctx, "pmm_fault_model3", strModel3, []int{0, 1, 2}, len(strModel3))
}

func pmmFault2Redis5() {
	ctx := context.Background()
	database.Redis.Del(ctx, "pmm_fault_model5")
	faultList := []int{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C,
	}
	strModel5 := Utils.IntArr2StringArr(faultList)
	Utils.ArrayHig2Redis(ctx, "pmm_fault_model5", strModel5, []int{0, 1, 2, 3, 4}, len(strModel5))
}


// pmmAlarm2Redis
// PMM AlarmType 告警类型枚举 Redis
func pmmAlarm2Redis3() {
	ctx := context.Background()
	database.Redis.Del(ctx, "pmm_alarm_model3")
	alarmList := []int{
		0x01, 0x02, 0x03,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e,
		0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28,
	}
	strModel := Utils.IntArr2StringArr(alarmList)
	Utils.ArrayLow2Redis(ctx, "pmm_alarm_model3", strModel, []int{0, 1, 2}, len(strModel))
}

func pmmAlarm2Redis5() {
	ctx := context.Background()
	database.Redis.Del(ctx, "pmm_alarm_model5")
	alarmList := []int{
		0x01, 0x02, 0x03,
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e,
		0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28,
	}
	strModel := Utils.IntArr2StringArr(alarmList)
	Utils.ArrayHig2Redis(ctx, "pmm_alarm_model5", strModel, []int{0, 1, 2, 3, 4}, len(strModel))
}

// ohpAlarm2Redis
// OHP AlarmType 告警类型枚举 Redis
func ohpAlarm2Redis3() {
	ctx := context.Background()
	database.Redis.Del(ctx, "ohp_alarm_model3")
	alarmList := []int{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06,
		0x10, 0x11, 0x12,
		0x20, 0x21, 0x22, 0x23,
	}
	strModel := Utils.IntArr2StringArr(alarmList)
	Utils.ArrayLow2Redis(ctx, "ohp_alarm_model3", strModel, []int{0, 1, 2}, len(strModel))
}

func ohpAlarm2Redis5() {
	ctx := context.Background()
	database.Redis.Del(ctx, "ohp_alarm_model5")
	alarmList := []int{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06,
		0x10, 0x11, 0x12,
		0x20, 0x21, 0x22, 0x23,
	}
	strModel := Utils.IntArr2StringArr(alarmList)
	Utils.ArrayHig2Redis(ctx, "ohp_alarm_model5", strModel, []int{0, 1, 2, 3, 4}, len(strModel))
}

