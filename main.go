package main

import (
	"github.com/ssimunic/gosensors"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	host   = os.Getenv("SENSOR_COLLECTOR_HOST")
	period = os.Getenv("SENSOR_PERIOD")
)

func init() {
	if host == "" {
		host = "https://peaceful-shelf-99858.herokuapp.com"
	}
	if period == "" {
		period = "10"
	}
}

func main() {
	p, err := strconv.Atoi(period)
	if err != nil {
		log.Fatalf("invalid period value: %s\n", p)
	}
	sendStats()
	ticker := time.NewTicker(time.Duration(p) * time.Second)
	for range ticker.C {
		sendStats()
	}
}

func sendStats() {
	sensors, err := gosensors.NewFromSystem()

	var body string
	if err == nil {
		body = sensors.String()
	} else {
		body = `{"error":"` + err.Error() + `"}`
	}

	ip, err := getIP()
	if err != nil {
		log.Printf("[ERROR] failed to get IP: %v\n", err)
		return
	}

	req, err := http.NewRequest("POST", host+"/sensor/"+ip.String(), strings.NewReader(body))
	if err != nil {
		log.Printf("[ERROR] failed to create request: %v\n", err)
		return
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[ERROR] failed to send sensor data: %v\n", err)
		return
	}
	if res.StatusCode >= 300 {
		body, _ := ioutil.ReadAll(res.Body)
		log.Printf("[ERROR] failed to send sensor data: unexpected response code (%d): %s", res.StatusCode, string(body))
		return
	}
	log.Printf("[INFO] sensor data sent: %s\n", body)
}

func getIP() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP, nil
}
