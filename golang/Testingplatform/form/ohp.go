package form

type OHPMessage struct {
	ProjectType string `json:"project_type"`
	MessageCode string `json:"message_code"`
	MessageBody string `json:"message_body"`
}

type Meter struct {
	Id uint32 `json:"id"`
}