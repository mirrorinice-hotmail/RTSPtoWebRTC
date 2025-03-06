package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"sync"

	"github.com/hashicorp/go-version"
	"github.com/liip/sheriff"
)

/*"github.com/imdario/mergo"*/
var configFile string = "config.json"

// Config global
var gConfig ConfigST

// ConfigST struct
type ConfigST struct {
	mutex         sync.RWMutex
	HttpServer    HttpServerST        `json:"http_server" groups:"config"`
	Server        ServerST            `json:"server" groups:"config"`
	Streams       map[string]StreamST `json:"streams" groups:"config"`
	Streams_extra map[string]StreamST `json:"streams_extra" groups:"config"`
	LastError     error
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
}

func (cfg *ConfigST) GetICEServers() []string {
	cfg.mutex.Lock()
	defer cfg.mutex.Unlock()
	return cfg.Server.ICEServers
}

func (cfg *ConfigST) GetICEUsername() string {
	cfg.mutex.Lock()
	defer cfg.mutex.Unlock()
	return cfg.Server.ICEUsername
}

func (cfg *ConfigST) GetICECredential() string {
	cfg.mutex.Lock()
	defer cfg.mutex.Unlock()
	return cfg.Server.ICECredential
}

func (cfg *ConfigST) GetWebRTCPortMin() uint16 {
	cfg.mutex.Lock()
	defer cfg.mutex.Unlock()
	return cfg.Server.WebRTCPortMin
}

func (cfg *ConfigST) GetWebRTCPortMax() uint16 {
	cfg.mutex.Lock()
	defer cfg.mutex.Unlock()
	return cfg.Server.WebRTCPortMax
}

func (cfg *ConfigST) loadConfig() {
	data, err := os.ReadFile(configFile)
	if err == nil {
		err = json.Unmarshal(data, &cfg)
		if err != nil {
			log.Fatalln(err)
		}
		for iUuid, tmpStream := range cfg.Streams {
			tmpStream.Cl = make(map[string]avQueue)
			tmpStream.Uuid = iUuid
			cfg.Streams[iUuid] = tmpStream
		}
	} else {
		addr := flag.String("listen", "8083", "HTTP host:port")
		udpMin := flag.Int("udp_min", 0, "WebRTC UDP port min")
		udpMax := flag.Int("udp_max", 0, "WebRTC UDP port max")
		iceServer := flag.String("ice_server", "", "ICE Server")
		flag.Parse()

		cfg.HttpServer.HTTPPort = *addr
		cfg.Server.WebRTCPortMin = uint16(*udpMin)
		cfg.Server.WebRTCPortMax = uint16(*udpMax)
		if len(*iceServer) > 0 {
			cfg.Server.ICEServers = []string{*iceServer}
		}

		cfg.Streams = make(map[string]StreamST)
	}
}

// ClientDelete Delete Client
func (in_cfgdata *ConfigST) SaveConfig() error {
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
	data, err := sheriff.Marshal(options, in_cfgdata)
	if err != nil {
		return err
	}
	//data := in_cfgdata
	JsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile("config_backup.json", JsonData, 0644)
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
