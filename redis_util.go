package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/robertkrimen/otto"
	"gopkg.in/redis.v5"
)

const (
	OWM = iota
	AX
)

var remoteServers = []int{
	OWM,
	AX,
}

// TODO: may use pool instead?
func NewClient() (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB

	})

	return client, nil
}

func Set(client *redis.Client, info, weather string) error {
	err := client.Set(info, weather, 0).Err()
	if err != nil {
		glog.Errorf("Fail to get weather for %s from remote server", info)
		return err
	} else {
		glog.Infof("Get weather from remote %s : %s", info, weather)
		return nil
	}
}

func main() {
	flag.Parse()
	defer glog.Flush()

	client, err := NewClient()
	if err != nil {
		glog.Errorf("Fail to create client for redis")
		return
	}

	// try to get weather info locally first
	// if there is no result from local, then get weather info from remote
	// get from only open weather API first to save resources
	// if it fails, use go routines to make it fast
	info := "101040100" // weathercode_[1d|7d|23d]
	// weathercode: id for city's weather info
	// [1d|7d|23d]: weather info of 1day/7day/23day
	// so, client gives city name, query xml to get weather code
	// get weather info from html by weather code
	// store weather info in redis,
	// when query with city name again, get code first
	// then get from redis
	// when to update redis, I guess the interval should be fixed
	// [20, 23, 02, 05, 08, 11, 14, 17, 20] -> need double check
	// then, when get query, first compared with current hour,
	// we can figure out whether it's time to update redis info

	// 01日20时,n01,多云,12℃,无持续风向,微风,0
	// 01日23时,n02,阴,11℃,无持续风向,微风,0
	// 02日02时,n02,阴,10℃,无持续风向,微风,0
	// 02日05时,n02,阴,10℃,无持续风向,微风,0
	// 02日08时,d02,阴,10℃,无持续风向,微风,3
	// 02日11时,d02,阴,11℃,无持续风向,微风,3
	// 02日14时,d02,阴,12℃,无持续风向,微风,3
	// 02日17时,d02,阴,12℃,无持续风向,微风,3
	// 02日20时,n02,阴,12℃,无持续风向,微风,0

	val, err := client.Get(info).Result()
	if err == redis.Nil {
		glog.Warningf("Weather for %s is nil, need to get from remote", info)

		if weather, err := GetWeather(info, remoteServers[0]); err != nil {
			glog.Warningf("Fail to get weather from %s, please try other servers", remoteServers[0])

			for i := 1; i < len(remoteServers); i++ {
				go func(client *redis.Client) {
					weather, err := GetWeather(info, remoteServers[i])
					if err != nil {
						glog.Errorf("Fail to get weather from %s", remoteServers[i])
						return
					} else {
						// timeout := 0 // TODO: need more logic
						if err := Set(client, info, weather); err != nil {
							// TODO: do nothing since it's in loop
							// return err
						}
					}
				}(client)
			}
		} else {
			if err := Set(client, info, weather); err != nil {
				// TODO: do nothing since it's in loop
				// return err
			}
			val = weather
		}
	} else if err != nil {
		glog.Errorf("Fail to get weather from client %s", err)
	}

	// TODO: cancel ongoing routines if there is already success info get
	fmt.Printf("Weather info for %s is %s\n", info, val)
	glog.Infof("Weather info for %s is %s\n", info, val)
}
