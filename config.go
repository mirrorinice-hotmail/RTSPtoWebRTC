package main

import (
	"encoding/json"
	"flag"
	"log"
	"net"
	"os"
	"sync"
)

/*"github.com/imdario/mergo"*/
var configFile string = "config.json"

// Config global
var gConfig ConfigST

// ConfigST struct
type ConfigST struct {
	mutex      sync.RWMutex
	Dbms       DbmsST       `json:"dbms" groups:"config"`
	HttpServer HttpServerST `json:"http_server" groups:"config"`
	Server     ServerST     `json:"server" groups:"config"`
	LastError  error
}

type DbmsST struct {
	Type      string `json:"type" groups:"config"`
	Host      string `json:"host" groups:"config"`
	Port      int    `json:"port" groups:"config"`
	User      string `json:"user" groups:"config"`
	Pass      string `json:"pass" groups:"config"`
	Dbname    string `json:"dbname" groups:"config"`
	TableName string `json:"tablename" groups:"config"`
}

// ServerST struct
type ServerST struct {
	ICEServers    []string `json:"ice_servers" groups:"config"`
	ICEUsername   string   `json:"ice_username" groups:"config"`
	ICECredential string   `json:"ice_credential" groups:"config"`
	WebRTCPortMin uint16   `json:"webrtc_port_min" groups:"config"`
	WebRTCPortMax uint16   `json:"webrtc_port_max" groups:"config"`
}

// Http Server struct
type HttpServerST struct {
	HTTPPort string `json:"http_port" groups:"config"`
	HTTPHost string `json:"http_host" groups:"config"`
}

func (obj *ConfigST) GetICEServers() []string {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	return obj.Server.ICEServers
}

func (obj *ConfigST) GetICEUsername() string {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	return obj.Server.ICEUsername
}

func (obj *ConfigST) GetICECredential() string {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	return obj.Server.ICECredential
}

func (obj *ConfigST) GetWebRTCPortMin() uint16 {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	return obj.Server.WebRTCPortMin
}

func (obj *ConfigST) GetWebRTCPortMax() uint16 {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	return obj.Server.WebRTCPortMax
}

func (obj *ConfigST) loadConfig() {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	data, err := os.ReadFile(configFile)
	if err == nil {
		err = json.Unmarshal(data, &obj)
		if err != nil {
			log.Fatalln(err)
		}
	} else {
		addr := flag.String("listen", "8083", "HTTP host:port")
		udpMin := flag.Int("udp_min", 0, "WebRTC UDP port min")
		udpMax := flag.Int("udp_max", 0, "WebRTC UDP port max")
		iceServer := flag.String("ice_server", "", "ICE Server")
		flag.Parse()

		obj.HttpServer.HTTPPort = *addr
		obj.Server.WebRTCPortMin = uint16(*udpMin)
		obj.Server.WebRTCPortMax = uint16(*udpMax)
		if len(*iceServer) > 0 {
			obj.Server.ICEServers = []string{*iceServer}
		}
	}

	if obj.HttpServer.HTTPHost == "" {
		obj.HttpServer.HTTPHost = GetFirstLocalIp()
	}
}

/*
	func (obj *ConfigST) SaveConfig() error {
		// log.WithFields(logrus.Fields{
		// 	"module": "config",
		// 	"func":   "NewStreamCore",
		// }).Debugln("Saving configuration to", configFile)
		v2, err := version.NewVersion("2.0.0")
		if err != nil {
			return err
		}

		options := &sheriff.Options{
			Groups:     []string{"config"},
			ApiVersion: v2,
		}
		data, err := sheriff.Marshal(options, obj)
		if err != nil {
			return err
		}
		//data := obj
		JsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return err
		}

		err = os.WriteFile(configFile, JsonData, 0644)
		if err != nil {
			// log.WithFields(logrus.Fields{
			// 	"module": "config",
			// 	"func":   "SaveConfig",
			// 	"call":   "WriteFile",
			// }).Errorln(err.Error())
			return err
		}

		return nil
	}
*/
func GetFirstLocalIp() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Println("GetFirstLocalIp().. Error:", err)
		return ""
	}
	// Iterate through each address to print the IP
	for _, addr := range addrs {
		// Convert network address to IPNet
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			// IPv4 check (excluding IPv6)
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}
	return ""
}
