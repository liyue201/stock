package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	logging "github.com/ipfs/go-log/v2"
	"github.com/liyue201/golib/xsignal"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

var log = logging.Logger("main")

type Record struct {
	Day   string `json:"day"`
	Close string `json:"close"`
}

var rejectErr = errors.New("reject")

func getKline(symbol string) (records []Record, err error) {
	url := strings.Replace(urlFmt, "sz002095", symbol, -1)
	resp, err := http.Get(url)
	if err != nil {
		log.Errorf("failed to get: %v", err.Error())
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Infof("failed to get: %v, %v", symbol, resp.StatusCode)
		return nil, rejectErr
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("failed to read data: %v", err.Error())
		return nil, err
	}
	//log.Infof("%v", string(data))
	err = json.Unmarshal(data, &records)
	if err != nil {
		log.Errorf("failed to unmarshal: %v", err.Error())
		return records, err
	}
	return records, nil
}

func convert(records []Record) []float64 {
	var list []float64
	for _, v := range records {
		t, err := time.Parse("2006-01-02 15:04:05", v.Day)
		if err != nil {
			log.Errorf("failed to parse time: %v", err)
			continue
		}
		if t.Hour() != 15 {
			continue
		}
		price, _ := strconv.ParseFloat(v.Close, 64)
		list = append(list, price)
	}
	return list
}

func getAvgPrice(prices []float64, day, back int) float64 {
	sum := 0.0
	ln := len(prices)
	for i := 0; i < day; i++ {
		sum += prices[ln-i-1-back]
	}
	return sum / float64(day)
}

func getAvgKLines(symbol string) ([]float64, error) {
	records, err := getKline(symbol)
	if err != nil {
		return nil, rejectErr
	}
	prices := convert(records)
	//log.Infof("%v: %v", symbol, prices)
	if len(prices) < 55+7 {
		return nil, nil
	}
	n := 7
	avgPrices := make([]float64, n)
	for i := 0; i < n; i++ {
		avgPrices[n-i-1] = getAvgPrice(prices, 55, i)
	}
	//log.Infof("avg %v: %v", symbol, avgPrices)
	return avgPrices, nil
}

func isTurnUp(avgPrices []float64) bool {
	if len(avgPrices) != 7 {
		return false
	}
	if avgPrices[0] > avgPrices[4] && avgPrices[5] > avgPrices[4] && avgPrices[6] > avgPrices[5] {
		return true
	}
	return false
}

var (
	goodStocks []string
	thisStocks []string
	mark       map[int64]struct{}
	mut        sync.Mutex
)

func handleSymbols(ctx context.Context, symbols []string) {
	mut.Lock()
	thisStocks = make([]string, 0)
	mut.Unlock()

	var bigSymbols []string
	for i, symbol := range symbols {
		select {
		case <-ctx.Done():
			return
		default:
		}
		time.Sleep(time.Second / 10)
		var value float64
		for loop := 0; loop < 5; loop++ {
			v, err := getValue(symbol)
			if err == rejectErr {
				time.Sleep(time.Minute * 5)
			} else {
				value = v
				break
			}
		}
		log.Infof("value %v, %v : %v", i, symbol, value)
		if value < 300 {
			continue
		}
		bigSymbols = append(bigSymbols, symbol)
	}
	log.Infof("bigSymbol num: %v", len(bigSymbols))
	for i, symbol := range bigSymbols {
		select {
		case <-ctx.Done():
			return
		default:
		}
		time.Sleep(time.Second * 2)
		log.Infof("handle %v: %v", i, symbol)
		var avgKline []float64
		for loop := 0; loop < 5; loop++ {
			kine, err := getAvgKLines(symbol)
			if err == rejectErr {
				time.Sleep(time.Minute * 5)
			} else {
				avgKline = kine
				break
			}
		}
		if isTurnUp(avgKline) {
			log.Infof("found %v", symbol)
			mut.Lock()
			thisStocks = append(thisStocks, symbol)
			mut.Unlock()
		}
	}
	mut.Lock()
	goodStocks = thisStocks
	mut.Unlock()
}

func run(ctx context.Context) {

	mark := make(map[int64]struct{})
	go func() {
		ticker := time.NewTicker(time.Hour)

		h := time.Now().Unix() / 60 / 60
		handleSymbols(ctx, gSymbols)
		mark[h] = struct{}{}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if time.Now().Hour() == 8 {
					h := time.Now().Unix() / 60 / 60
					if _, ok := mark[h]; !ok {
						handleSymbols(ctx, gSymbols)
						mark[h] = struct{}{}
					}
				}
			}
		}
	}()
}

func getGoodStocks(c *gin.Context) {
	ret := make(map[string]string)
	mut.Lock()
	defer mut.Unlock()
	if len(goodStocks) > 0 {
		for _, s := range goodStocks {
			ret[s] = gName[s]
		}
	} else {
		for _, s := range thisStocks {
			ret[s] = gName[s]
		}
	}

	c.IndentedJSON(http.StatusOK, ret)
}

func runHttp(port int) {
	engine := gin.Default()
	engine.GET("/api/stocks", getGoodStocks)
	addr := fmt.Sprintf("0.0.0.0:%d", port)
	httpServer := &http.Server{Addr: addr, Handler: engine}
	go func() {
		httpServer.ListenAndServe()
	}()
}

func main() {
	port := flag.Int("port", 9561, "--port")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	run(ctx)
	runHttp(*port)
	xsignal.Wait()
}
