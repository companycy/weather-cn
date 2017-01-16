package main

import (
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	// "reflect"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/golang/glog"
	"gopkg.in/redis.v5"
)

// <china dn="nay">
// 	<city quName="黑龙江" pyName="heilongjiang" cityname="哈尔滨" state1="53" state2="53" stateDetailed="霾" tem1="-19" tem2="-10" windState="西北风转西风小于3级"/>
// 	<city quName="吉林" pyName="jilin" cityname="长春" state1="1" state2="0" stateDetailed="多云转晴" tem1="-13" tem2="-4" windState="西南风小于3级"/>
type CountryWeather struct {
	XMLName       xml.Name `xml:"china"`
	CitiesWeather []struct {
		XMLName xml.Name `xml:"city"`
		QuName  string   `xml:"quName,attr"`
		PyName  string   `xml:"pyName,attr"` // only useful info for this xml
		// Cityname      string   `xml:"cityname,attr"`
		// StateDetailed string   `xml:"stateDetailed,attr"`
		// WindState     string   `xml:"windState,attr"`
		// State1        int      `xml:"state1,attr"`
		// State2        int      `xml:"state2,attr"`
		// Tem1          int      `xml:"tem1,attr"`
		// Tem2          int      `xml:"tem2,attr"`
	} `xml:"city"`
}

// http://flash.weather.com.cn/wmaps/xml/china.xml
var citiesPyMap = make(map[string]string, 0)

func loadCityPy() error {
	// use native xml in case there is strange py, like:
	// sanxi -> 陕西
	glog.Infof("load city pinyin...")

	xmlFile, err := os.Open("weather-china.xml")
	if err != nil {
		glog.Errorf(err.Error())
		return err
	}
	defer xmlFile.Close()

	xmlData, err := ioutil.ReadAll(xmlFile)
	if err != nil {
		glog.Errorf(err.Error())
		return err
	}

	var cw CountryWeather
	if err := xml.Unmarshal(xmlData, &cw); err != nil {
		glog.Errorf(err.Error())
		return err
	}

	for i := 0; i < len(cw.CitiesWeather); i++ {
		city := cw.CitiesWeather[i]
		citiesPyMap[city.QuName] = city.PyName
	}
	return nil
}

func getWeatherFromRemote(cityPy string) (*map[string]interface{}, error) {
	url := fmt.Sprintf("http://flash.weather.com.cn/wmaps/xml/%s.xml", cityPy)
	// TODO: may need to retry
	resp, err := http.Get(url)
	if err != nil {
		glog.Errorf("Failed to get url %s", url)
		return nil, err
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	weather := ProvinceWeather{}
	xml.Unmarshal(buf.Bytes(), &weather)
	// fmt.Println(url, weather)

	// <heilongjiang dn="nay">
	// <city cityX="186.35" cityY="83.6" cityname="大兴安岭" centername="大兴安岭" fontColor="FFFFFF" pyName="daxinganling" state1="1" state2="1" stateDetailed="多云" tem1="-28" tem2="-8" temNow="-20" windState="北风小于3级转西北风3-4级" windDir="西北风" windPower="2级" humidity="67%" time="22:20"
	// url="101050701"/>
	ret := make(map[string]interface{})
	for i := 0; i < len(weather.RegionsWeather); i++ {
		ret[weather.RegionsWeather[i].PyName] = weather.RegionsWeather[i]
		// fmt.Println(weather.RegionsWeather[i])
	}

	return &ret, nil
}

func getTime() (string, error) {
	now := time.Now()
	hour := now.Hour()
	log.Println("hour", hour)

	weather := "cloudy"
	return weather, nil
}

// <heilongjiang dn="nay">
// 	<city cityX="186.35" cityY="83.6" cityname="大兴安岭" centername="大兴安岭" fontColor="FFFFFF" pyName="daxinganling" state1="1" state2="1" stateDetailed="多云" tem1="-28" tem2="-8" temNow="-20" windState="北风小于3级转西北风3-4级" windDir="西北风" windPower="2级" humidity="67%" time="22:20" url="101050701"/>
type RegionWeather struct {
	// XMLName       xml.Name `xml:"city"`
	CityX         float64 `xml:"cityX,attr"`
	CityY         float64 `xml:"cityY,attr"`
	Cityname      string  `xml:"cityname,attr"`
	PyName        string  `xml:"pyName,attr"`
	State1        int     `xml:"state1,attr"`
	State2        int     `xml:"state2,attr"`
	StateDetailed string  `xml:"stateDetailed,attr"`
	Tem1          int     `xml:"tem1,attr"`
	Tem2          int     `xml:"tem2,attr"`
	TemNow        int     `xml:"temNow,attr"`
	WindState     string  `xml:"windState,attr"`
	WindDir       string  `xml:"windDir,attr"`
	WindPower     string  `xml:"windPower,attr"`
	Humidity      string  `xml:"humidity,attr"`
	Time          string  `xml:"time,attr"`
}

type ProvinceWeather struct {
	// XMLName        xml.Name `xml:"cn"`
	RegionsWeather []RegionWeather `xml:"city"`
}

const Separator = "|"               // TODO
const expiration = time.Duration(0) // TODO

func redisSet(client *redis.Client, key string, weather map[string]interface{}) (string, error) {
	glog.Infof("redis set %v", weather)
	var str string
	for k, v := range weather {
		rw, ok := v.(RegionWeather)
		if !ok {
			continue
		}
		// fmt.Printf("%v\n", rw)

		weatherStr := strconv.FormatFloat(rw.CityX, 'f', 2, 32) +
			"_" + strconv.FormatFloat(rw.CityY, 'f', 2, 32) +
			"_" + rw.Cityname + "_" + rw.PyName +
			"_" + strconv.Itoa(rw.State1) + "_" + strconv.Itoa(rw.State2) + "_" + rw.StateDetailed +
			"_" + strconv.Itoa(rw.Tem1) + "_" + strconv.Itoa(rw.Tem2) + "_" + strconv.Itoa(rw.TemNow) +
			"_" + rw.WindState + "_" + rw.WindDir + "_" + rw.WindPower +
			"_" + rw.Humidity + "_" + strings.Replace(rw.Time, ":", "", 1)

		glog.Infof("weather str: %s\n", weatherStr)
		str += k + ":" + weatherStr + "|"
	}

	str = str[:len(str)-1]
	err := client.Set(key, str, expiration).Err()
	if err != nil {
		glog.Errorf("Fail to set %s in redis", key)
		return "", err
	} else {
		glog.Infof("Result str: %s", str)
		return str, nil
	}
}

var redisClient *redis.Client

func init() {
	flag.Parse()
	defer glog.Flush()

	// init redis client
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	if err := loadCityPy(); err != nil {
		glog.Errorf(err.Error())
	}
}

func needUpdate(val string) bool {
	day := val[:2]
	_, _, curDay := time.Now().Date()
	i, err := strconv.Atoi(day)
	if err != nil {
		glog.Error("fail to get day %s", err)
		return true
	}

	glog.Info("from redis ", i, " cur day ", curDay)
	if i != curDay {
		return true
	}
	return false
}

// {'jiangsu' : { 'xuzhou': {}, 'wuxi': {} }, 'beijing': {'haidian': {}, 'chaoyang': {}},  }
// jiangsu_rt is xuzhou:xx_xx_xx|wuxi:xx_xx_xx|...
func weatherToJson(val string, ret *map[string]map[string]string) error {
	glog.Infof("weatherStr: %s", val)

	sss := strings.Split(val, "|")
	for i := 0; i < len(sss); i++ {
		ss := strings.Split(sss[i], ":")
		if len(ss) != 2 {
			glog.Errorf("sth wrong when split weather str %s", ss)
			continue
		}

		// hdr := ss[0] // xuzhou, use chinese instead
		v := strings.Split(ss[1], "_")
		if len(v) < 15 {
			glog.Errorf("sth wrong when split weather str %s", v)
			continue
		}

		// <city quName="黑龙江" pyName="heilongjiang" cityname="哈尔滨" state1="53" state2="53" stateDetailed="霾" tem1="-19" tem2="-10" windState="西北风转西风小于3级"/>
		m := make(map[string]string)
		j := 0
		m["cityX"] = v[j]
		j++
		m["cityY"] = v[j]
		j++
		cityName := v[j]
		m["cityName"] = v[j]
		j++
		m["pyName"] = v[j]
		j++
		m["state1"] = v[j]
		j++
		m["state2"] = v[j]
		j++
		m["stateDetailed"] = v[j]
		j++
		m["tem1"] = v[j]
		j++
		m["tem2"] = v[j]
		j++
		m["temNow"] = v[j]
		j++
		m["windState"] = v[j]
		j++
		m["windDir"] = v[j]
		j++
		m["windPower"] = v[j]
		j++
		m["humidity"] = v[j]
		j++
		m["time"] = v[j]
		j++

		(*ret)[cityName] = m // in case m[cityName] is not assigned
	}

	return nil
}

func realtimeWeatherHandler(req *restful.Request, resp *restful.Response) {
	cityName := req.QueryParameter("cn") // cityname
	glog.Infof("realtimeWeatherHandler %s...", cityName)

	ret := map[string]interface{}{
		"ret_code": 0,
	}
	var dstProvincesPyList []string
	if cityName == "全国" {
		for _, v := range citiesPyMap {
			dstProvincesPyList = append(dstProvincesPyList, v)
		}
	} else {
		if v, ok := citiesPyMap[cityName]; ok {
			dstProvincesPyList = append(dstProvincesPyList, v)
		} else {
			ret := map[string]interface{}{
				"ret_code": 5000,
				"err":      "Invalid province name",
			}
			resp.WriteAsJson(ret)
			return
		}
	}

	for i := 0; i < len(dstProvincesPyList); i++ {
		key := dstProvincesPyList[i] + "_rt" // py_rt, ex. jiangsu_rt
		glog.Infof("key %s", key)
		val, err := redisClient.Get(key).Result()
		ret2 := make(map[string]map[string]string)
		if err == redis.Nil { //  || needUpdate(val)
			glog.Infof("realtime weather for %s is nil, update from remote", key)
			weather, err := getWeatherFromRemote(dstProvincesPyList[i])
			if err != nil {
				ret := map[string]interface{}{
					"ret_code": 5000,
					"err":      "fail to get county weather",
				}
				resp.WriteAsJson(ret)
				return
			} else {
				weatherStr, err := redisSet(redisClient, key, *weather)
				if err != nil {
					// TODO
				}
				val = weatherStr
			}
		} else if err != nil {
			glog.Errorf("Fail to get weather from client %s", err)
		}

		err = weatherToJson(val, &ret2)
		if err != nil {
			// TODO:
		}
		ret[cityName] = ret2
	}

	resp.WriteAsJson(ret)
}

func main() {
	ws := new(restful.WebService)
	ws.Route(ws.GET("/v1/query/realtime").To(realtimeWeatherHandler))
	restful.Add(ws)
	http.ListenAndServe(":8081", nil)
}
