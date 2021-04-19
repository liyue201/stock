package main

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

var gSymbols []string
var gName map[string]string

const urlFmt = "http://money.finance.sina.com.cn/quotes_service/api/json_v2.php/CN_MarketData.getKLineData?symbol=sz002095&scale=60&ma=no&datalen=1023"
const valueUrlFmt = "http://qt.gtimg.cn/q=s_sz002095"

func init() {
	gName = make(map[string]string)
	f := func(rs []string, pre string) {
		for _, s := range rs {
			ss := strings.Split(s, "(")
			if len(ss) < 1 {
				continue
			}
			sss := strings.Split(ss[1], ")")
			symbol := pre + sss[0]
			gName[symbol] = ss[0]
			gSymbols = append(gSymbols, symbol)
		}
	}
	f(shangzheng, "sh")
	f(sz, "sz")
	f(chuangye, "sz")
	log.Infof("total: %v", len(gSymbols))
}


func getValue(symbol string) (float64, error ) {
	url := strings.Replace(valueUrlFmt, "sz002095", symbol, -1)
	resp, err := http.Get(url)
	if err != nil {
		log.Errorf("failed to get: %v", err.Error())
		return 0.0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Infof("failed to get: %v, %v", symbol, resp.StatusCode)
		return 0.0, rejectErr
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("failed to read data: %v", err.Error())
		return 0.0, err
	}

	ss := strings.Split(string(data), "=")
	if len(ss) < 2 {
		log.Errorf("fmt error: %v", string(data))
		return 0.0, errors.New("fmt error")
	}
	sss := strings.Split(ss[1], "~")
	if len(sss) < 10 {
		log.Errorf("fmt error: %v", string(ss[1]))
		return 0.0, errors.New("fmt error")
	}

	v := sss[9]

	value, err := strconv.ParseFloat(v, 64)
	if err != nil {
		log.Errorf("failed to parse: %v", err.Error())
		return 0.0, err
	}

	return value, nil
}