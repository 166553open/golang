package database

import (
	"TestingPlatform/Utils"
	"github.com/go-redis/redis"
)

func redis_connect() *redis.Client {
	redisConf, err := Utils.ReadFile("./conf/database.json", "redis")
	if err != nil {
		return nil
	}
	redisConfData := redisConf.(map[string]interface{})
	host := redisConfData["host"].(string)
	port := redisConfData["port"].(string)
	db := int(redisConfData["db"].(float64))
	password := redisConfData["password"].(string)
	return redis.NewClient(&redis.Options{
		Addr:               host+":"+port,
		Password:           password,
		DB:                 db,
	})
}
