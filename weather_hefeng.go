package main

import (
	// "encoding/json"
	"fmt"
	"net/http"
	"bytes"

	"github.com/bitly/go-simplejson"

)

const (
	ApiKey = "a20f3450826743a28bdab38822b5d2d1" // api key from hefeng
)

// refer to hefeng-nodejs and below link for API info:
// https://www.hefeng.com/doc/request.aspx
const (
	HefengHost       = "https://free-api.heweather.com/v5/"
	HefengCurVersion = "v5"
	HefengPath       = HefengCurVersion + "/"

	CurPath = HefengHost + "/now"
	// ForcastPath = HefengHost + HefengPath + "/forecast.json"
	// SearchPath  = HefengHost + HefengPath + "/search.json"
	// HistoryPath = HefengHost + HefengPath + "/history.json"
)

// type BasicInfo struct {
// 	City, Cnty, Id, Prov  string
// 	Lat, Lon float64
// 	Update struct {
// 		Loc, Utc string
// 	}
// }

// type Response struct {
	
// }

// type CurWeather struct {
// 	Resp Response`json:"HeWeather5"`
// }



func main() {
	// var cur CurWeather

	// curl -d "city=beijing&key=a20f3450826743a28bdab38822b5d2d1"  "https://free-api.heweather.com/v5/now"
	city := "beijing"
	url := fmt.Sprintf("%s?key=%s&city=%s", CurPath, ApiKey, city)
	js, err := getJson(url)
	if err != nil {
		fmt.Println("Failed to get current weather %s", err)
	}

	weatherInfo := js.Get("HeWeather5").GetIndex(0)
	basicInfo := weatherInfo.Get("basic")
	lon := basicInfo.Get("lon")
	lat := 

	nowWeather := weatherInfo.Get("now")
	status := weatherInfo.Get("status")
	fmt.Println(basicInfo)
	fmt.Println(nowWeather)
	fmt.Println(status)
}

func getJson(url string) (*simplejson.Json, error) {
	r, err := http.Get(url)
	if err != nil {
		fmt.Println("Failed to get url %s", url)
		return nil, err
	}
	defer r.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(r.Body)
	
	js, err := simplejson.NewJson(buf.Bytes())
	if err != nil {
		fmt.Println("Failed to get url %s", url)
		return nil, err
	}
	return js, nil
}
