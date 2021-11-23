package form

import "errors"

type PMMMessage struct {
	ProjectType string `json:"project_type"`
	MessageCode string `json:"message_code"`
	MessageBody string `json:"message_body"`
}


type PMMRegister struct {
	Id uint32 `json:"id"`						// 主线程id
}

type PMMRegisterPower struct {
	Id uint32 `json:"id"`				// 主线程id
	State int `json:"state"`			// 功率矩阵模块自检状态
}

type PMMRegisterAdModule struct {
	Id uint32 `json:"id"`				// 主线程id
	IsAlarm bool `json:"is_alarm"`		// 是否设置模块的alarm故障
	AlarmLength int `json:"alarm_length"` // 设置模块的alarm故障长度
}

type PMMRegisterHeartBeat struct {
	Id uint32 `json:"id"`						// 主线程id
	Interval uint32 `json:"interval"`			// 心跳间隔
	HeartbeatCtr int64 `json:"heartbeat_ctr"`	// 心跳计数
}


type PMMContactorHeartBeat struct {
	Id uint32 `json:"id"`						// 模块线程id
	Interval uint32 `json:"interval"`			// 心跳间隔
	HeartbeatCtr int64 `json:"heartbeat_ctr"`	// 心跳计数
	AlarmState []int `json:"alarm_state"`		// 故障码
}

type PMMContactorRT struct {
	Id uint32 `json:"id"`						// 模块线程id
}

type PMMRTInfo struct {
	Status bool `json:"status"`							// RT状态（是否发送故障）
	Id uint32 `json:"id"`						// 枪Id
	PushCnt uint32 `json:"push_cnt"`					// 推送计数
	Interval uint32 `json:"interval"`	// 心跳间隔
}

type PMMConfig struct {
	Id uint32 `json:"id"`
	Interval uint32 `json:"interval"`	// 修改心跳间隔
	HeartbeatCtr int64 `json:"heartbeat_ctr"`		 	// 修改心跳计数
}

func (m *PMMRTInfo) VerifyVCIRTInfo () error {
	if m.Interval == 0 {
		return errors.New("预期时间必须大于零")
	}
	return nil
}