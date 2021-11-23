package database

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
	NoPassReason	string `json:"no_pass_reason" xorm:"comment('未通过原因')"`
	IsDelete        int    `json:"is_delete" xorm:"comment('是否删除')"`
	TestDate        string `json:"test_date" xorm:"comment('测试日期')"`
	CurrentTime     int64  `json:"current_time"`
	CreatedAt       string `json:"created_at" xorm:"<-"`
	UpdatedAt       string `json:"updated_at" xorm:"<-"`
}
/*
CREATE TABLE IF NOT EXISTS test_collection(
	`id` int(11) not null auto_increment primary key,
	`project_id` int(11) not null comment "项目名称 test_project id 关联键",
	`message_type` int(1) not null comment "消息类型 1 普通数据（注册）2 心跳数据（心跳）3 突发数据（突发）",
	`input_message` text not null comment "测试项目输入",
	`output_message` text not null comment "测试输出消息体（可输出多个）",
	`expected_message` text not null comment "预期输出消息体",
	`time_cost`	int(64) not null comment "测试用例耗时，单位（ms）",
	`is_pass` int(1) not null comment "测试用例是否通过",
	`no_pass_reason` varchar(255) comment "未通过原因",
	`is_delete` int(1) not null comment "是否删除",
	`test_date` varchar(50) not null comment "测试日期",
	`current_time` int(64) not null comment "当前时间",
	`created_at` timestamp ,
	`updated_at` timestamp
)
*/

type TestMessage struct {
	ID 			int `json:"id" xorm:"pk autoincr"`
	ProjectId 	int `json:"project_id" xorm:"comment('项目id')"`
	ProjectName string `json:"project_name" xorm:"comment('项目名称')"`
	MessageName string `json:"message_name" xorm:"comment('消息体名称')"`
	MessageCode int `json:"message_code" xorm:"comment('消息体code')"`
	MessageBody string `json:"message_body" xorm:"comment('消息体内容')"`
}
/*
CREATE TABLE IF NOT EXISTS test_message(
	`id` int(11) not null auto_increment primary key,
	`project_id` int(11) not null comment "项目id",
	`message_code` int(1) not null comment "消息体code",
	`project_name` varchar(50) not null comment "项目名称",
	`message_name` varchar(50) not null comment "消息体名称",
	`message_body` text not null comment "消息体内容",
	`created_at` timestamp ,
	`updated_at` timestamp
)
*/

type TestProjects struct {
	ID 			int `json:"id" xorm:"pk autoincr"`
	ProjectName string `json:"project_name"`
}
/*
CREATE TABLE IF NOT EXISTS test_projects(
	`id` int(11) not null auto_increment primary key,
	`project_name` varchar(50) not null comment "项目名称",
	`created_at` timestamp ,
	`updated_at` timestamp
)
*/