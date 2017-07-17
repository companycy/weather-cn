# weather-cn

	// city := "beijing"
	// if err := getWeatherByCityName(city); err != nil {
	// 	fmt.Println(err)
	// }

	// if err := getWeatherByRegionName(); err != nil {
	// 	fmt.Println(err)
	// }

	// info, err := getCountyInfo(cityName)
	// if err != nil {
	// 	fmt.Println(err)
	// } else {
	// 	fmt.Println(info)
	// }

	// if _, err := getWeatherFromRemote(info.weatherCode, WCC); err != nil {
	// 	fmt.Println(err)
	// }


curl "http://119.254.100.75:8081/v1/query/realtime?cn=江苏" 

curl "http://119.254.100.75:8080/v1/query/forcast?cn=苏州" 

