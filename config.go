package main

import (
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/deepch/vdk/codec/h264parser"

	"github.com/deepch/vdk/av"
)

// Config global
var Config = loadConfig()

// ConfigST struct
type ConfigST struct {
	mutex      sync.RWMutex
	HttpServer HttpServerST        `json:"http_server"`
	Server     ServerST            `json:"server"`
	Streams    map[string]StreamST `json:"streams"`
	LastError  error
}

// ServerST struct
type ServerST struct {
	//	HTTPPort      string   `json:"http_port"`
	ICEServers    []string `json:"ice_servers"`
	ICEUsername   string   `json:"ice_username"`
	ICECredential string   `json:"ice_credential"`
	WebRTCPortMin uint16   `json:"webrtc_port_min"`
	WebRTCPortMax uint16   `json:"webrtc_port_max"`
}

// Http Server struct
type HttpServerST struct {
	HTTPPort string `json:"http_port"`
}

// StreamST struct
type StreamST struct {
	URL          string `json:"url"`
	Status       bool   `json:"status"`
	OnDemand     bool   `json:"on_demand"`
	DisableAudio bool   `json:"disable_audio"`
	Debug        bool   `json:"debug"`
	RunLock      bool   `json:"-"`
	Codecs       []av.CodecData
	Cl           map[string]viewer
}

type viewer struct {
	c chan av.Packet
}

func (element *ConfigST) RunAllStream() {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	for iUuid, iStream := range element.Streams {
		if iStream.RunLock {
			continue
		}
		iStream.RunLock = true
		element.Streams[iUuid] = iStream
		log.Println("RunStream of all", iUuid)
		go RTSPWorkerLoop(iUuid, iStream.URL, iStream.OnDemand, iStream.DisableAudio, iStream.Debug)
	}
}

func (element *ConfigST) RunIFNotRun(uuid string) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	if iStream, ok := element.Streams[uuid]; ok {
		if iStream.OnDemand && !iStream.RunLock {
			log.Println("RunIFNotRun", uuid)
			iStream.RunLock = true
			element.Streams[uuid] = iStream
			go RTSPWorkerLoop(uuid, iStream.URL, iStream.OnDemand, iStream.DisableAudio, iStream.Debug)
		}
	}
}

func (element *ConfigST) RunStream(uuid string) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	iStream, ok := element.Streams[uuid]
	if !ok || iStream.RunLock {
		return
	}

	log.Println("RunStream", uuid)
	iStream.RunLock = true
	element.Streams[uuid] = iStream
	go RTSPWorkerLoop(uuid, iStream.URL, iStream.OnDemand, iStream.DisableAudio, iStream.Debug)

}

func (element *ConfigST) Runlock(uuid string, onOff bool) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	iStream, ok := element.Streams[uuid]
	if !ok || iStream.RunLock {
		return
	}

	if iStream.RunLock {
		iStream.RunLock = onOff
		element.Streams[uuid] = iStream
	}

}

func (element *ConfigST) HasViewer(uuid string) bool {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	if tmp, ok := element.Streams[uuid]; ok && len(tmp.Cl) > 0 {
		return true
	}
	return false
}

func (element *ConfigST) GetICEServers() []string {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	return element.Server.ICEServers
}

func (element *ConfigST) GetICEUsername() string {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	return element.Server.ICEUsername
}

func (element *ConfigST) GetICECredential() string {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	return element.Server.ICECredential
}

func (element *ConfigST) GetWebRTCPortMin() uint16 {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	return element.Server.WebRTCPortMin
}

func (element *ConfigST) GetWebRTCPortMax() uint16 {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	return element.Server.WebRTCPortMax
}

func loadConfig() *ConfigST {
	var tmp ConfigST
	data, err := os.ReadFile("config.json")
	if err == nil {
		err = json.Unmarshal(data, &tmp)
		if err != nil {
			log.Fatalln(err)
		}
		for i, v := range tmp.Streams {

			v.Cl = make(map[string]viewer)
			tmp.Streams[i] = v
		}
	} else {
		addr := flag.String("listen", "8083", "HTTP host:port")
		udpMin := flag.Int("udp_min", 0, "WebRTC UDP port min")
		udpMax := flag.Int("udp_max", 0, "WebRTC UDP port max")
		iceServer := flag.String("ice_server", "", "ICE Server")
		flag.Parse()

		tmp.HttpServer.HTTPPort = *addr
		tmp.Server.WebRTCPortMin = uint16(*udpMin)
		tmp.Server.WebRTCPortMax = uint16(*udpMax)
		if len(*iceServer) > 0 {
			tmp.Server.ICEServers = []string{*iceServer}
		}

		tmp.Streams = make(map[string]StreamST)
	}
	return &tmp
}

func (element *ConfigST) cast(uuid string, pck av.Packet) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	for _, v := range element.Streams[uuid].Cl {
		if len(v.c) < cap(v.c) {
			v.c <- pck
		}
	}
}

func (element *ConfigST) ext(suuid string) bool {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	_, ok := element.Streams[suuid]
	return ok
}

func (element *ConfigST) coAd(suuid string, codecs []av.CodecData) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	t := element.Streams[suuid]
	t.Codecs = codecs
	element.Streams[suuid] = t
}

func (element *ConfigST) coGe(suuid string) []av.CodecData {
	for i := 0; i < 100; i++ {
		element.mutex.RLock()
		tmp, ok := element.Streams[suuid]
		element.mutex.RUnlock()
		if !ok {
			return nil
		}
		if tmp.Codecs != nil {
			//TODO Delete test
			for _, codec := range tmp.Codecs {
				if codec.Type() == av.H264 {
					codecVideo := codec.(h264parser.CodecData)
					if codecVideo.SPS() != nil && codecVideo.PPS() != nil && len(codecVideo.SPS()) > 0 && len(codecVideo.PPS()) > 0 {
						//ok
						//log.Println("Ok Video Ready to play")
					} else {
						//video codec not ok
						log.Println("Bad Video Codec SPS or PPS Wait")
						time.Sleep(50 * time.Millisecond)
						continue
					}
				}
			}
			return tmp.Codecs
		}
		time.Sleep(50 * time.Millisecond)
	}
	return nil
}

func (element *ConfigST) clAd(suuid string) (string, chan av.Packet) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	cuuid := pseudoUUID()
	ch := make(chan av.Packet, 100)
	element.Streams[suuid].Cl[cuuid] = viewer{c: ch}
	return cuuid, ch
}

func (element *ConfigST) list() (string, []string) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	var res []string
	var fist string
	for k := range element.Streams {
		if fist == "" {
			fist = k
		}
		res = append(res, k)
	}
	return fist, res
}
func (element *ConfigST) clDe(suuid, cuuid string) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	delete(element.Streams[suuid].Cl, cuuid)
}

func pseudoUUID() (uuid string) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	uuid = fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return
}
