package main

import (
	"fmt"
	"net/http"
	// "io/ioutil"
	"bytes"
	"strings"
	// "net/url"
	// "io"
	// "os"
	"encoding/json"
)

type Weather []map[string]string

func main() {
	err := getCityCode()
	if err != nil {
		fmt.Println(err)
	}
}

// 从中国天气网获取city code, 可能还是直接读取xml文件更好
// refer to link: https://my.oschina.net/cart/blog/189839
func getCityCode() error {
	// curl -e "http://www.weather.com.cn/forecast/index.shtml" "http://toy1.weather.com.cn/search?cityname=%E5%8D%97%E4%BA%AC"
	urlStr := "http://toy1.weather.com.cn/search"
	r, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return err
	}

	q := r.URL.Query()
	q.Add("cityname", "北京")
	r.URL.RawQuery = q.Encode()
	r.Header.Add("Referer", "http://www.weather.com.cn/forecast/index.shtml")
	// r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	// r.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
	// r.Header.Set("Content-Encoding", "gzip")
	// r.Header.Set("Accept-Encoding", "gzip, deflate")

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// below are 3 methods to get string from response's Body

	// method 1 from http://stackoverflow.com/questions/23967638/ioutil-readallresponse-body-blocks-forever-golang
	// content, _ := ioutil.ReadAll(resp.Body)
	// newStr := string(content)

	// method 2 from http://golangcode.com/convert-io-readcloser-to-a-string/
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	newStr := buf.String()

	res := Weather{}
	json.Unmarshal([]byte(newStr[1:len(newStr)-1]), &res)
	for i := 0; i < len(res); i++ {
		for k, v := range res[i] {
			fmt.Println(k, strings.Split(v, "~"))
		}
	}

	// method 3 from https://gist.github.com/ijt/950790
	// _, err = io.Copy(os.Stdout, resp.Body)
	// if err != nil {
	// 	fmt.Println(err)
	// }

	return nil
}
