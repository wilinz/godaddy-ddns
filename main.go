package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/weili71/go-filex"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

type Config struct {
	Domain     string `json:"domain,omitempty"`
	Key        string `json:"key,omitempty"`
	Secret     string `json:"secret,omitempty"`
	RecordType string `json:"record_type,omitempty"`
	Name       string `json:"name,omitempty"`
	IpServer   string `json:"ip_server,omitempty"`
}

type Address struct {
	IP   string `json:"IP"`
	Port int    `json:"Port"`
}

type UpdateDnsBody struct {
	Data string `json:"data"` //ip
	Name string `json:"name"`
	TTL  int    `json:"ttl"`
	Type string `json:"type"`
}

func main() {
	programDir := filex.NewFile(os.Args[0]).Parent()

	templateConfigFile := filex.NewFile1(programDir, "config_template.json")
	if !templateConfigFile.IsExist() {
		configTemplate, _ := json.Marshal(Config{
			Domain:     "xxx.com",
			Key:        "godaddy key",
			Secret:     "godaddy secret",
			RecordType: "A",
			Name:       "www",
			IpServer:   "http://yyy.com",
		})

		var out bytes.Buffer
		err := json.Indent(&out, configTemplate, "", "\t")
		if err != nil {
			log.Panicln(err)
			return
		}

		err = templateConfigFile.Write(out.Bytes(), 0777)
		if err != nil {
			log.Panicln(err)
			return
		}
	}

	configFile := filex.NewFile1(programDir, "config.json")
	configBytes, err := configFile.ReadAll()
	if err != nil {
		log.Panicln(err)
		return
	}

	var config Config
	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		log.Panicln(err)
		return
	}

	run(config)
}

func run(config Config) {
	for {
		oldIP := "0.0.0.0"
		oldIPAddr, err := net.ResolveIPAddr("ip", config.Name+"."+config.Domain)
		if err == nil {
			oldIP = oldIPAddr.String()
		}
		newIP, err1 := getNewAddress(config)
		if err1 != nil {
			time.Sleep(time.Minute)
			continue
		}
		if newIP == oldIP {
			log.Println("ip不变")
		} else {
			fmt.Println("ip变化，正在更新dns记录")
			err := updateDnsRecord(config, newIP)
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Printf("dns更新成功，旧地址：%s,新地址：%s\n", oldIP, newIP)
			}
		}
		time.Sleep(time.Minute)
	}
}

func getNewAddress(config Config) (string, error) {
	resp, err := http.Get(config.IpServer)
	if err != nil {
		return "", err
	}
	respBytes, err1 := ioutil.ReadAll(resp.Body)
	if err1 != nil {
		return "", err1
	}
	newAddress := Address{}
	err2 := json.Unmarshal(respBytes, &newAddress)
	if err2 != nil {
		return "", err2
	}
	return newAddress.IP, nil
}

func updateDnsRecord(config Config, ip string) error {
	url := fmt.Sprintf("https://api.godaddy.com/v1/domains/%s/records/%s/%s", config.Domain, config.RecordType, config.Name)
	updateDnsBody := []UpdateDnsBody{{
		Data: ip,
		Name: "home",
		TTL:  600,
		Type: "A",
	}}

	updateDnsBodyBytes, _ := json.Marshal(updateDnsBody)
	req, _ := http.NewRequest("PUT", url, bytes.NewReader(updateDnsBodyBytes))
	req.Header.Set("Authorization", fmt.Sprintf("sso-key %s:%s", config.Key, config.Secret))
	req.Header.Set("Content-Type", "application/json")
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode == 200 {
		return nil
	}

	respBytes, err1 := ioutil.ReadAll(resp.Body)
	if err1 != nil {
		return err1
	}
	return errors.New(string(respBytes))
}
