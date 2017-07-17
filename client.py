#!/usr/bin/env python
#-*- coding:utf-8 -*-
import requests
import sys
import simplejson as jsmod

def main(args):
    # http://119.254.100.75:8080/v1/query
    ip = "119.254.100.75"
    url = "http://%s:8080/v1/query" % (ip)
    ret = requests.post(url, data={
        # "cn": "重庆",
        "cn": "aaa",
    })
    obj = jsmod.loads(ret.text, encoding='utf-8')
    # read as below
    # {
    #    u'weather1d': u'02\u65e520\u65f6,n01,\u591a\u4e91,13\u2103,\u65e0\u6301\u7eed\u98ce\u5411,\u5fae\u98ce,0|02\u65e523\u65f6,n02,\u9634,12\u2103,\u65e0\u6301\u7eed\u98ce\u5411,\u5fae\u98ce,0|03\u65e502\u65f6,n02,\u9634,11\u2103,\u65e0\u6301\u7eed\u98ce\u5411,\u5fae\u98ce,0|03\u65e505\u65f6,n02,\u9634,11\u2103,\u65e0\u6301\u7eed\u98ce\u5411,\u5fae\u98ce,0|03\u65e508\u65f6,d02,\u9634,11\u2103,\u65e0\u6301\u7eed\u98ce\u5411,\u5fae\u98ce,3|03\u65e511\u65f6,d02,\u9634,12\u2103,\u65e0\u6301\u7eed\u98ce\u5411,\u5fae\u98ce,3|03\u65e514\u65f6,d02,\u9634,13\u2103,\u65e0\u6301\u7eed\u98ce\u5411,\u5fae\u98ce,3|03\u65e517\u65f6,d02,\u9634,13\u2103,\u65e0\u6301\u7eed\u98ce\u5411,\u5fae\u98ce,3|03\u65e520\u65f6,n02,\u9634,13\u2103,\u65e0\u6301\u7eed\u98ce\u5411,\u5fae\u98ce,0',
    #    u'ret_code': 0
    # }
    if obj["ret_code"] == 0:
        weather1d = obj["weather1d"]
        # print weather1d
        weather_info = weather1d.split("|")
        for hourly_weather in weather_info:
            print hourly_weather
    else:
        err = obj["err"]
        print "err when get weather: ", err

if __name__ == "__main__":
    main(sys.argv[1:])
