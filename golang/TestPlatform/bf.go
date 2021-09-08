package main

import (
	"fmt"
	"goWork/database"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func main()  {
	resRedis := database.Redis.Ping()
	resMysql, err := database.MySQL.QueryString("select * from users")
	fmt.Println(resRedis, resMysql, err)
	http.HandleFunc("/copy", func(w http.ResponseWriter, r *http.Request) {
		body := r.Body
		defer body.Close()

		fmt.Println(r.Body)
		res, err := ioutil.ReadAll(r.Body)
		fmt.Println(string(res))
		if err != nil {
			fmt.Printf("this is error with io:%s", err.Error())
			return
		}
		//net.Dial("tcp", "127.0.0.1")

		if _, err := io.Copy(os.Stdout, r.Body); err != nil {
			log.Fatal(err)
		}


	})

	http.ListenAndServe(":8080", nil)

}

