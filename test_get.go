package main

import (
	"bytes"
	"log"
	"net/http"
	// "encoding/xml"
	// "flag"
	// "golang.org/x/net/html"
	// "io/ioutil"
	// "os"
	// "reflect"
	// "strconv"
	// "strings"
	// "time"
	// "github.com/emicklei/go-restful"
	// "github.com/golang/glog"
	// "github.com/robertkrimen/otto"
	// "gopkg.in/redis.v5"
)

func main() {
	// ws := new(restful.WebService)
	// ws.Route(ws.GET("/v1/query/forcast").To(weatherHandler))
	// restful.Add(ws)
	// http.ListenAndServe(":8080", nil)

	url := "http://www.weather.com.cn/weather/101221601.shtml"
	// log.Println("url to get from remote srv", url)
	// resp, err := http.Get(url)
	// if err != nil {
	// 	glog.Errorf("failed to get weather from remote server: %v", err.Error())
	// 	return
	// }
	// defer resp.Body.Close()
	// buf := new(bytes.Buffer)
	// buf.ReadFrom(resp.Body)
	// // log.Println(buf.String())
	// // s := strings.Replace(buf.String(), "^M", "", -1)

	client := &http.Client{}
	// resp, err := client.Get(url)
	req, err := http.NewRequest("GET", url, nil)
	req.Close = true
	resp, err := client.Do(req)
	if err != nil {
		// whatever
	}
	defer resp.Body.Close()

	// response, err := ioutil.ReadAll(resp.Body)
	// if err != nil {
	// 	// Whatever
	// }
	// log.Println("new resp", response)

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	log.Println(buf.String())
	// s := strings.Replace(buf.String(), "^M", "", -1)

}
