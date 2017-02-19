package main

import (
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"golang.org/x/net/html"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/golang/glog"
	"github.com/robertkrimen/otto"
	"gopkg.in/redis.v5"
)

// <china dn="nay">
// 	<city quName="黑龙江" pyName="heilongjiang" cityname="哈尔滨" state1="53" state2="53" stateDetailed="霾" tem1="-19" tem2="-10" windState="西北风转西风小于3级"/>
// 	<city quName="吉林" pyName="jilin" cityname="长春" state1="1" state2="0" stateDetailed="多云转晴" tem1="-13" tem2="-4" windState="西南风小于3级"/>

func printWeather(m map[string]interface{}) {
	for k, v := range m {
		switch k {
		case day1: // today
			log.Println(reflect.TypeOf(v))
			if s, ok := v.([]string); ok {
				for i := 0; i < len(s); i++ {
					log.Println("  ", s[i])
				}
			}
		case day7: // week?
			log.Println(reflect.TypeOf(v))
			if ss, ok := v.([][]string); ok {
				for i := 0; i < len(ss); i++ {
					for j := 0; j < len(ss[i]); j++ {
						log.Println("  ", ss[i][j])
					}
				}
			}
		case day23: // month?
			log.Println(reflect.TypeOf(v))
			if ss, ok := v.([][]string); ok {
				for i := 0; i < len(ss); i++ {
					for j := 0; j < len(ss[i]); j++ {
						log.Println("  ", ss[i][j])
					}
				}
			}
		}
	}
}

type WeatherSrvNotSupportedError struct {
}

func (e WeatherSrvNotSupportedError) Error() string {
	// TODO: should include srv info
	return fmt.Sprintf("Current weather srv is unsupported: %v", e)
}

type EmptyWeatherInfoError struct {
}

func (e EmptyWeatherInfoError) Error() string {
	return fmt.Sprintf("Weather is empty from remote: %v", e)
}

type InvalidTokenError struct {
}

func (e InvalidTokenError) Error() string {
	return fmt.Sprintf("Invalid token error in html: %v", e)
}

type WeatherError struct {
}

func (e WeatherError) Error() string {
	return fmt.Sprintf("Failed to get weather from remote: %v", e)
}

func getWeatherFromRemote(cityCode string, srvCode int) (*map[string]interface{}, error) {
	glog.Infof("get weather for city %s from remote %v", cityCode, srvCode)

	if srvCode != WCC { // TODO
		glog.Errorf("only www.weather.com.cn supported now")
		return nil, WeatherSrvNotSupportedError{}
	}

	htmlStr, err := getHtmlFromRemote(cityCode, srvCode)
	if err != nil || htmlStr == "" {
		glog.Errorf("fail to get html from remote %v", cityCode)
		return nil, EmptyWeatherInfoError{}
	}

	s := strings.Replace(htmlStr, "^M", "", -1)
	buf := bytes.NewBufferString(s)
	z := html.NewTokenizer(strings.NewReader(buf.String()))
	for {
		tt := z.Next()
		switch {
		// case tt == html.EndTagToken:
		// 	t := z.Token()
		// 	isScript := t.Data == "script"
		// 	if !isScript {
		// 		continue
		// 	}
		// 	fmt.Println("here goes script end ", t.String())
		// case tt == html.StartTagToken:
		// 	t := z.Token()
		// 	isScript := t.Data == "script"
		// 	if !isScript {
		// 		continue
		// 	}
		// 	fmt.Println("here goes script start ", t.String())
		// }
		// default:
		// 	fmt.Println("neither end nor script, go on... ")
		case tt == html.ErrorToken: // End of the document, we're done
			return nil, InvalidTokenError{}

		case tt == html.TextToken:
			t := z.Token()
			// isScript := t.Data == "script"
			// if !isScript {
			// 	continue
			// }
			c := t.String()
			keyword := "var hour3data"
			if !strings.Contains(c, keyword) {
				continue
			}

			// parse js
			c = strings.TrimSpace(c)
			c = strings.Replace(c, "&#34;", "\"", -1) // may be bug?

			var err error
			vm := otto.New()
			vm.Run(c)
			hour3data, err := vm.Get("hour3data")
			if err != nil {
				glog.Errorf("fail to get hour3data, %v", err.Error())
				continue
			}

			if !hour3data.IsObject() {
				glog.Errorf("hour3data is not obj, %v", err.Error())
				continue
			}

			// glog.Infof("hour3data to export: %v", hour3data)
			obj, err := hour3data.Export()
			if err != nil {
				glog.Errorf("fail to export hour3data, %v", err.Error())
				continue
			}

			// glog.Infof("hour3data obj to convert: %v", obj)
			m, ok := obj.(map[string]interface{})
			if !ok {
				glog.Errorf("fail to conver obj, %v", err.Error())
				continue
			}

			// printWeather(m)

			return &m, nil
		}
	}

	glog.Errorf("fail to get weather from remote")
	return nil, WeatherError{}
}

// <China>
//   <province id="01" name="北京">
//     <city id="0101" name="北京">
//       <county id="010101" name="北京" weatherCode="101010100"/>
//       <county id="010102" name="海淀" weatherCode="101010200"/>

type CityCode struct {
	XMLName   xml.Name `xml:"China"`
	Provinces []struct {
		XMLName xml.Name `xml:"province"`
		Id      string   `xml:"id,attr"`
		Name    string   `xml:"name,attr"`
		Cities  []struct {
			XMLName  xml.Name `xml:"city"`
			Id       string   `xml:"id,attr"`
			Name     string   `xml:"name,attr"`
			Counties []struct {
				XMLName     xml.Name `xml:"county"`
				Id          string   `xml:"id,attr"`
				Name        string   `xml:"name,attr"`
				WeatherCode string   `xml:"weatherCode,attr"`
			} `xml:"county"`
		} `xml:"city"`
	} `xml:"province"`
}

var cityInfoSet *map[string]CityInfo
var countyInfoSet *map[string]CountyInfo

type CityInfo struct {
	id, name    string
	countyNames []string
}

type CountyInfo struct {
	id, name, weatherCode string
}

func loadCitycode() (*map[string]CityInfo, *map[string]CountyInfo, error) {
	glog.Infof("load city code...")

	xmlFile, err := os.Open("weather-cn.xml")
	if err != nil {
		glog.Errorf("fail to open xml file %v", err.Error())
		return nil, nil, err
	}
	defer xmlFile.Close()

	xmlData, err := ioutil.ReadAll(xmlFile)
	if err != nil {
		glog.Errorf("fail to read xml file %v", err.Error())
		return nil, nil, err
	}

	var cc CityCode
	if err := xml.Unmarshal(xmlData, &cc); err != nil {
		glog.Errorf("fail to unmarshal xml file %v", err.Error())
		return nil, nil, err
	}

	cityInfoSet := make(map[string]CityInfo)
	countyInfoSet := make(map[string]CountyInfo)
	for i := 0; i < len(cc.Provinces); i++ {
		province := cc.Provinces[i]
		for j := 0; j < len(province.Cities); j++ {
			city := province.Cities[j]

			cityInfo := CityInfo{
				id:   city.Id,
				name: city.Name,
			}

			for k := 0; k < len(city.Counties); k++ {
				county := city.Counties[k]
				cityInfo.countyNames = append(cityInfo.countyNames, county.Name)
				// fmt.Println(county)

				countyInfoSet[county.Name] = CountyInfo{
					id:          county.Id,
					name:        county.Name,
					weatherCode: county.WeatherCode,
				}
			}
			cityInfoSet[city.Name] = cityInfo
		}
	}
	return &cityInfoSet, &countyInfoSet, nil
}

type CountryInfoError struct {
}

func (e CountryInfoError) Error() string {
	return fmt.Sprintf("Failed to get count info: %v", e)
}

func getCountyInfo(cityName string) (*CountyInfo, error) {
	glog.Infof("get county code %s", cityName)

	if info, ok := (*countyInfoSet)[cityName]; ok {
		return &info, nil
	} else {
		if info2, ok := (*cityInfoSet)[cityName]; ok { // relay to sub county
			glog.Infof("try to relay to sub county %s", cityName)
			newName := info2.countyNames[0]
			if info3, ok := (*countyInfoSet)[newName]; ok {
				return &info3, nil
			}
		}

		return nil, CountryInfoError{}
	}
}

const Separator = "|"

type HandleWeatherError struct {
}

func (e HandleWeatherError) Error() string {
	return fmt.Sprintf("Failed to handle weather: %v", e)
}

func handleWeatherStr(s string, out map[string]interface{}) (string, error) {
	idx := strings.Index(s, ",")
	if idx != -1 {
		hdr := s[:idx]
		v := strings.Split(s[idx+1:], ",")
		if len(v) < 5 {
			glog.Errorf("sth wrong when split weather str %s", v)
			return "", HandleWeatherError{}
		}

		len := len(v[2]) - 3 // remove ending strange "℃,"
		v[2] = v[2][:len]
		if out != nil {
			m := make(map[string]string)
			m["var1"] = v[0]
			m["cloud"] = v[1]
			m["temp"] = v[2]
			m["wind1"] = v[3]
			m["wind2"] = v[4]
			// m["var2"] = v[5]
			out[hdr] = m
		}

		ret := hdr + ":" + strings.Join(v, "_") // time1:temp1_cloud1|time2:temp2_cloud2
		return ret, nil
	} else {
		return "", HandleWeatherError{} // TODO
	}
}

const expiration = time.Duration(0) // todo:

type RedisSetError struct {
}

func (e RedisSetError) Error() string {
	return fmt.Sprintf("Failed to set redis: %v", e)
}

func redisSet(client *redis.Client, key string, weather map[string]interface{}) (string, error) {
	glog.Infof("redis set value for key: %s", key)

	for k, v := range weather {
		if !strings.HasSuffix(key, k) {
			continue
		}

		switch k {
		case day1: // today
			// glog.Info(reflect.TypeOf(v))
			if s, ok := v.([]string); ok {
				var weatherStr string
				for i := 0; i < len(s); i++ {
					s1, _ := handleWeatherStr(s[i], nil) // TODO
					weatherStr += s1 + Separator
				}
				weatherStr = weatherStr[:len(weatherStr)-1]
				err := client.Set(key, weatherStr, expiration).Err()
				if err != nil {
					glog.Errorf("fail to set %s in redis", key)
					return "", err
				} else {
					glog.Infof("set key %s : value %s", key, weatherStr)
					return weatherStr, nil
				}
			}
		case day7: // week?
			// glog.Info(reflect.TypeOf(v))
			if ss, ok := v.([][]string); ok {
				// fmt.Println("total len ", len(ss))
				// m := make(map[string][]string)
				var weatherStr string
				for i := 0; i < len(ss); i++ {
					var s1 string
					for j := 0; j < len(ss[i]); j++ {
						s, _ := handleWeatherStr(ss[i][j], nil) // TODO
						s1 += s + Separator
					}
					weatherStr += s1
				}
				weatherStr = weatherStr[:len(weatherStr)-1] // remove ending "|"

				err := client.Set(key, weatherStr, expiration).Err()
				if err != nil {
					glog.Errorf("fail to set %s in redis", key)
					return "", err
				} else {
					glog.Infof("set key %s: value %s", key, weatherStr)
					return weatherStr, nil
				}
			}
		case day23: // month, not used currently
			// glog.Infof(reflect.TypeOf(v))
			if ss, ok := v.([][]string); ok {
				for i := 0; i < len(ss); i++ {
					for j := 0; j < len(ss[i]); j++ {
						glog.Infof("month weather:  ", ss[i][j])
					}
				}
			}
		}
	}

	return "", RedisSetError{}
}

const (
	WCC = iota
	OWM
	AX
)

var remoteServers = []int{
	WCC,
	OWM,
	AX,
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

	cityIS, countyIS, err := loadCitycode()
	if err != nil {
		glog.Errorf("fail to load city code %v", err.Error())
	}
	cityInfoSet = cityIS
	countyInfoSet = countyIS
}

func getWeatherFromRemote2(info *CountyInfo, key string) {
	for i := 1; i < len(remoteServers); i++ {
		go func(client *redis.Client) {
			weather, err :=
				getWeatherFromRemote(info.weatherCode, remoteServers[i])
			if err != nil {
				glog.Errorf("fail to get weather from %s", remoteServers[i])
				// ret := map[string]interface{}{
				// 	"ret_code": 5000,
				// 	"err":      "fail to get county code",
				// }
				// resp.WriteAsJson(ret)
				return
			} else {
				if _, err := redisSet(client, key, *weather); err != nil {
					// TODO: do nothing since it's in loop
					// return err
				}
			}
		}(redisClient)
	} // end of for loop
}

const (
	day1  = "1d"
	day7  = "7d"
	day23 = "23d"
)

var days = []string{
	// day1,
	day7,
	// day23,
}

func needUpdate(val string) bool {
	day := val[:2]
	dayInRedis, err := strconv.Atoi(day)
	if err != nil {
		glog.Errorf("fail to get day %s", err)
		return true
	}

	_, _, curDay := time.Now().Date()
	glog.Infof("time in redis: %v while cur day: %v", dayInRedis, curDay)
	if dayInRedis != curDay {
		return true
	}
	return false
}

// 1d: ["":{}, ]
// ["":{}, ]
// 7d: [08日: [15时: {}, 24时: {}],  09日: [15时: {}, 24时: {}], 10日:[15时: {}, 24时: {}],  ]
// [08日: [15时: {}, 24时: {}],  09日: [15时: {}, 24时: {}], 10日:[15时: {}, 24时: {}],  ]
// 101190401_7d is 15日20时:n01_多云_5_东北风_微风|15日23时:n01_多云_1_东北风_3-4级|16日02时:n01_多云_1_东北风_微风
func weatherToJson2(val string, ret *[]map[string][]map[string]map[string]string) error {
	glog.Infof("source val: %v dst json: %v", val, ret)

	sss := strings.Split(val, "|")
	for i := 0; i < len(sss); i++ {
		ss := strings.Split(sss[i], ":")
		if len(ss) != 2 {
			glog.Errorf("sth wrong when split weather str %s", ss)
			continue
		}

		hdr := ss[0]
		v := strings.Split(ss[1], "_")
		if len(v) < 5 {
			glog.Errorf("sth wrong when split weather str %s", v)
			continue
		}

		// [{08日: [{15时: {}}, {24时: {}}]},  {09日: [{15时: {}}, {24时: {}}]}  ]
		m := make(map[string]string)
		m["var1"] = v[0]
		m["weather1"] = v[1]
		m["temp1"] = v[2]
		m["wind1"] = v[3]
		m["wind2"] = v[4]
		// m["var2"] = v[5]
		// out[hdr] = m
		key := hdr[:2] + "日"
		if len(*ret) == 0 {
			glog.Infof("json is empty, add key to json %s", key)
			*ret = append(*ret, map[string][]map[string]map[string]string{
				key: make([]map[string]map[string]string, 0),
			})
		}

		vv, ok := (*ret)[len(*ret)-1][key]
		if !ok {
			glog.Infof("json has no key %s, fill it", key)
			*ret = append(*ret, map[string][]map[string]map[string]string{
				key: make([]map[string]map[string]string, 0),
			})
		}

		vv, ok = (*ret)[len(*ret)-1][key]
		if !ok {
			glog.Errorf("fail to append element %v", ret)
			continue
		} else {
			// log.Println(vv)
		}

		vv = append(vv, map[string]map[string]string{
			hdr: m,
		})
		(*ret)[len(*ret)-1][key] = vv
		// log.Println((*ret)[len(*ret)-1][key])
	}

	// log.Println(ret)
	return nil
}

func getHtmlFromRemote(cityCode string, srvCode int) (string, error) {
	// url1 := "http://www.weather.com.cn/weather/101221601.shtml"
	url1 := "http://www.weather.com.cn/weather1d/" + cityCode + ".shtml"
	// log.Println("url to get from remote srv", url1)
	resp1, err := http.Get(url1)
	if err != nil {
		glog.Errorf("failed to get weather from remote server: %v", err.Error())
		return "", err
	}
	defer resp1.Body.Close()
	buf1 := new(bytes.Buffer)
	buf1.ReadFrom(resp1.Body)
	if !strings.Contains(buf1.String(), "<!-- empty -->") { // empty html content {}
		return buf1.String(), nil
	}
	log.Printf("get empty html: %v from remote srv: %v\n", url1, buf1.String())

	var cmd = exec.Command("curl", url1)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Start(); err != nil {
		return "", err
	}
	err = cmd.Wait()
	if err != nil {
		glog.Errorf("curl cmd also fails: %v, %v", string(output.Bytes()), err)
		return "", err
	}

	s := string(output.Bytes())
	log.Printf("cmd: %v output: %v\n", cmd, s)
	return s, nil
}

func weatherHandler(req *restful.Request, resp *restful.Response) {
	cityName := req.QueryParameter("cn") // cityname
	glog.Infof("weatherHandler for %s...", cityName)

	// try to get weather info locally first
	// if there is no result from local, then get weather info from remote
	// get from only open weather API first to save resources
	// if it fails, use go routines to make it fast

	info, err := getCountyInfo(cityName)
	if info != nil {
		glog.Infof("id: %s, code: %s, name: %s", info.id, info.weatherCode, info.name)
	} else {
		glog.Errorf("fail to get county code %s", err)
		ret := map[string]interface{}{
			"ret_code": 5000,
			"err":      "fail to get county code",
		}
		resp.WriteAsJson(ret)
		return
	}

	ret := map[string]interface{}{
		"ret_code": 0,
	}
	for i := 0; i < len(days); i++ { // only 1d/7d is required
		key := info.weatherCode + "_" + days[i] // weathercode_[1d|7d|23d]
		// weathercode: id for city's weather info
		// [1d|7d|23d]: weather info of 1day/7day/23day

		// so, client gives city name, query xml to get weather code
		// get weather info from html by weather code
		// store weather info in redis,
		// when query with city name again, get code first
		// then get from redis
		// when to update redis, I guess the interval should be fixed
		// [20, 23, 02, 05, 08, 11, 14, 17, 20]
		// [08, 11, 14, 17, 20, 23, 02, 05, 08]
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

		var ret2 []map[string][]map[string]map[string]string
		val, err := redisClient.Get(key).Result()
		// glog.Infof("in redis, key : %v, val: %v", key, val)

		if err == redis.Nil || needUpdate(val) {
			glog.Infof("weather for %s is nil or outdated, update from remote", key)

			weather, err := getWeatherFromRemote(info.weatherCode, remoteServers[0])
			if err != nil {
				glog.Infof("fail to get from %s, try other servers", remoteServers[0])
				continue
				// TODO: not necessary now
				// getWeatherFromRemote2(info, key)
			} else {
				// glog.Infof("val to set in redis: %v", weather)
				weatherStr, err := redisSet(redisClient, key, *weather)
				if err != nil {
					// TODO: do nothing since it's in loop
					// return err
				}
				val = weatherStr

				// err = weatherToJson2(val, &ret2) // prepare for json
				// if err != nil {
				// 	// TODO:
				// }
				// log.Println(ret2)
			}

		} else if err != nil {
			glog.Errorf("fail to get weather from client %s", err)
			continue
		}

		err = weatherToJson2(val, &ret2)
		if err != nil {
			// TODO:
		}
		// log.Println(ret2)

		ret[days[i]] = ret2
		glog.Infof("weather info to return for %s is %s\n", days[i], ret2)
	}

	resp.WriteAsJson(ret)
}

func main() {
	ws := new(restful.WebService)
	ws.Route(ws.GET("/v1/query/forcast").To(weatherHandler))
	restful.Add(ws)
	http.ListenAndServe(":8080", nil)
}
