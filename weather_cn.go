package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http"
)

type City struct {
	XMLName       xml.Name `xml:"city"`
	QuName        string   `xml:"quName,attr"`
	PyName        string   `xml:"pyName,attr"`
	Cityname      string   `xml:"cityname,attr"`
	StateDetailed string   `xml:"stateDetailed,attr"`
	WindState     string   `xml:"windState,attr"`
	State1        int      `xml:"state1,attr"`
	State2        int      `xml:"state2,attr"`
	Tem1          int      `xml:"tem1,attr"`
	Tem2          int      `xml:"tem2,attr"`
}

type WeatherInfo struct {
	XMLName  xml.Name `xml:"china"`
	CityList []City   `xml:"city"`
}

// no more json is provided, thus use xml instead
// http://flash.weather.com.cn/wmaps/xml/china.xml
func getWeather() error {
	url := "http://flash.weather.com.cn/wmaps/xml/china.xml"
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Failed to get url %s", url)
		return err
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	weather := WeatherInfo{}
	xml.Unmarshal(buf.Bytes(), &weather)
	for i := 0; i < len(weather.CityList); i++ {
		fmt.Println(weather.CityList[i])
	}

	return nil
}

func main() {
	err := getWeather()
	if err != nil {
		fmt.Println(err)
	}
}
