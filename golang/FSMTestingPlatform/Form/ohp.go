package form

import "errors"

type OHPMessage struct {
	ProjectType string `json:"project_type"`
	MessageCode string `json:"message_code"`
	MessageBody string `json:"message_body"`
}

type OHPConfig struct {
	Id uint32 `json:"id"`			// 订单流水线线程ID
}

type OHPRegister struct {
	Id	uint32 `json:"id"`			// 订单流水线线程ID
	State	int `json:"state"`		// 订单流水线模块自检状态
}

type OHPHeartBeat struct {
	Id uint32 `json:"id"`						// 模块线程id
	Interval uint32 `json:"interval"`			// 心跳间隔
	HeartbeatCtr int64 `json:"heartbeat_ctr"`	// 心跳计数
}

type OHPRTInfo struct {
	Status bool `json:"status"`							// RT状态（是否发送故障）
	GunId uint32 `json:"gun_id"`						// 枪Id
	PushCnt uint32 `json:"push_cnt"`					// 推送计数
	HeartBeatPeriod uint32 `json:"heart_beat_period"`	// 心跳间隔
}

func (m *OHPRTInfo) VerifyVCIRTInfo () error {
	if m.HeartBeatPeriod == 0 {
		return errors.New("预期时间必须大于零")
	}
	return nil
}