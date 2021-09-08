package database

import (
	"github.com/go-redis/redis"
	"xorm.io/xorm"
)
var MySQL *xorm.Engine
var Redis *redis.Client
var err error
func init () {
	MySQL, err = mysql_connect()
	if err != nil {

		return
	}

	Redis = redis_connect()
}
