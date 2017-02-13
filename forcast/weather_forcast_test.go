package main

import (
	"strings"
	"net/http"
	"bytes"
	"time"
	"log"

	"testing"

)

const (
)

func TestCityWeatherinfo(t *testing.T) {
	cityInfoSet, _, err := loadCitycode()
	if err != nil {
		t.Errorf("Failed to load city code for test")
	}

	errStrs := []string {
		"fail",
		"Empty",
		"error",
	}
	for cityName, _ := range *cityInfoSet {
		// t.Logf("City name: %s", cityName)
		log.Printf("City name: %v", cityName)
		url := "http://119.254.100.75:8080/v1/query/forcast?cn=" + cityName
		resp, err := http.Get(url)
		if err != nil {
			t.Fatalf(err.Error())
		}
		defer resp.Body.Close()

		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		for i := 0; i < len(errStrs); i++ {
			if strings.Contains(buf.String(), errStrs[i]) {
				t.Fatalf("Fail to get weather info for %v", cityName)
			}
		}
		time.Sleep(20 * time.Second)
	}
}
