package main

import (
	"crypto/tls"
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
		period = "30"
	}
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
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

	mac, err := getMAC()
	if err != nil {
		log.Printf("[ERROR] failed to get IP: %v\n", err)
		return
	}

	req, err := http.NewRequest("POST", host+"/sensor/"+mac, strings.NewReader(body))
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

// Based on https://www.socketloop.com/tutorials/golang-how-do-I-get-the-local-ip-non-loopback-address
func getMAC() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	var currentIP string
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			// check the address type and if it is not a loopback the display it
			if ipnet.IP.To4() != nil {
				currentIP = ipnet.IP.String()
			}
		}
	}
	// get all the system's or local machine's network interfaces
	var currentName string
	interfaces, _ := net.Interfaces()
	for _, iface := range interfaces {
		if addrs, err := iface.Addrs(); err == nil {
			for _, addr := range addrs {
				// only interested in the name with current IP address
				if strings.Contains(addr.String(), currentIP) {
					currentName = iface.Name
				}
			}
		}
	}
	netInterface, err := net.InterfaceByName(currentName)
	if err != nil {
		return "", nil
	}
	mac := netInterface.HardwareAddr
	// verify the MAC address can be parsed properly
	_, err = net.ParseMAC(mac.String())
	if err != nil {
		return "", err
	}
	return mac.String(), nil
}