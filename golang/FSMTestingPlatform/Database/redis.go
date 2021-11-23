package database

import (
	"github.com/go-redis/redis/v8"
)

// redisConnect
// ----------------------------------------------------------------------------
// 链接到redis库
// ----------------------------------------------------------------------------
func redisConnect() *redis.Client {
	redisConfData, err := readFile("./Conf/database.json", "redis")
	if err != nil {
		return nil
	}
	host := redisConfData["host"].(string)
	port := redisConfData["port"].(string)
	db := int(redisConfData["db"].(float64))
	password := redisConfData["password"].(string)
	return redis.NewClient(&redis.Options{
		Addr:     host + ":" + port,
		Password: password,
		DB:       db,
	})
}
