package form

import "errors"

type VCIMessage struct {
	ProjectType string `json:"project_type"`
	MessageCode string `json:"message_code"`
	MessageBody string `json:"message_body"`
}

func (m *VCIMessage) VerifyMainRegister () error {
	if m.ProjectType != "VCI" || m.MessageCode != "0" {
		return errors.New("参数格式不正确")
	}
	return nil
}

func (m *VCIMessage) VerifyMainHeart () error {
	if m.ProjectType != "VCI" || m.MessageCode != "2" {
		return errors.New("参数格式不正确")
	}
	return nil
}


type VCIGun struct {
	Gun uint32 `json:"gun"`
}

type FatherForm struct {
	Gun uint32 `json:"gun"`
}

type SonForm struct {
	Gun uint32 `json:"gun"`
}

