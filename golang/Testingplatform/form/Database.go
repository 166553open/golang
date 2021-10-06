package form

type SqlMessage struct {
	MessageCode string	`json:"message_code"`
	MessageName string	`json:"message_name"`
	MessageType int		`json:"message_type"`
}

type TestCollection struct {
	ID              int    `json:"id" xorm:"pk autoincr"`
	ProjectId       int    `json:"project_id" xorm:"comment('项目名称 test_project id 关联键')"`
	MessageType     int    `json:"message_type" xorm:"comment('消息类型\r\n1 普通数据（注册）\r\n2 心跳数据（心跳）\r\n3 突发数据（突发）')"`
	InputMessage    string `json:"input_message" xorm:"text comment('测试项目输入')"`
	OutputMessage   string `json:"output_message" xorm:"text comment('测试输出消息体（可输出多个）')"`
	ExpectedMessage string `json:"expected_message" xorm:"text comment('预期输出消息体')"`
	TimeCost        int64  `json:"time_cost" xorm:"comment('测试用例耗时，单位（ms）')"`
	IsPass          int    `json:"is_pass" xorm:"comment('测试用例是否通过')"`
	IsDelete        int    `json:"is_delete" xorm:"comment('是否删除')"`
	TestDate        string `json:"test_data" xorm:"comment('测试日期')"`
	CurrentTime     int64  `json:"current_time"`
	CreatedAt       string `json:"created_at" xorm:"<-"`
	UpdatedAt       string `json:"updated_at" xorm:"<-"`
}

type TestMessage struct {
	ID 			int `json:"id" xorm:"pk autoincr"`
	ProjectId 	int `json:"project_id" xorm:"comment('项目id')"`
	ProjectName string `json:"project_name" xorm:"comment('项目名称')"`
	MessageName string `json:"message_name" xorm:"comment('消息体名称')"`
	MessageCode int `json:"message_code" xorm:"comment('消息体code')"`
	MessageBody string `json:"message_body" xorm:"comment('消息体内容')"`
}

type TestProjects struct {
	ID 			int `json:"id" xorm:"pk autoincr"`
	ProjectName string `json:"project_name"`
}