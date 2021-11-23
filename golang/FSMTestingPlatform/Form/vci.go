package form

import "errors"

type VCIMessage struct {
	ProjectType string `json:"project_type"`
	MessageCode string `json:"message_code"`
	MessageBody string `json:"message_body"`
}

type VCIRegisterConfig struct {
	Id       uint32 `json:"id"`
	Interval uint32 `json:"interval"`
}

type VCIConfig struct {
	Id              uint32 `json:"id"`
	GunId           uint32 `json:"gun_id"`
	HeartBeatPeriod uint32 `json:"heart_beat_period"`
}

type VCIRegister struct {
	SelfCheckState int `json:"self_check_state"`
}

type VCIRegisterHeartBeat struct {
	GunId           uint32                   `json:"gun_id"`            // 充电线程枪Id
	HeartBeatCnt    uint32                   `json:"heart_beat_cnt"`    // 心跳计数
	HeartBeatPeriod uint32                   `json:"heart_beat_period"` // 心跳间隔
	TimeNow         uint32                   `json:"time_now"`          // 当前时间戳
	GunBasicState   vciRegisterGunBasicState `json:"gun_basic_state"`   //枪状态
}

type vciRegisterGunBasicState struct {
	LinkState       uint32 `json:"link_state"`       //    插枪状态,0表示空闲,1表示已插枪
	PositionedState uint32 `json:"positioned_state"` //    在位/归位状态,0表示未归位,1表示已归位
}

type ChargeHeartBeat struct {
	GunId           uint32 `json:"gun_id"`            // 充电线程枪Id
	HeartBeatCnt    int64  `json:"heart_beat_cnt"`    // 心跳计数
	HeartBeatPeriod uint32 `json:"heart_beat_period"` // 心跳间隔
}

type VCIRTInfo struct {
	Status          bool   `json:"status"`            // RT状态（是否发送故障）
	GunId           uint32 `json:"gun_id"`            // 枪Id
	PushCnt         uint32 `json:"push_cnt"`          // 推送计数
	HeartBeatPeriod uint32 `json:"heart_beat_period"` // 心跳间隔
}

func (m *VCIRTInfo) VerifyVCIRTInfo() error {
	if m.HeartBeatPeriod == 0 {
		return errors.New("预期时间必须大于零")
	}
	return nil
}
