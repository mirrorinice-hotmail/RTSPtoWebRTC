package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/deepch/vdk/codec/h264parser"
	"github.com/hashicorp/go-version"
	"github.com/liip/sheriff"

	"github.com/deepch/vdk/av"
)

/*"github.com/imdario/mergo"*/

var gStreamListInfo StreamListInfoST

// ///////////////////////////////////
type StreamListInfoST struct {
	mutex         sync.RWMutex
	Streams       StreamsMAP `json:"streams" groups:"config"`
	Streams_extra StreamsMAP `json:"streams_extra" groups:"config"`
	//LastError error
}

type StreamsMAP map[string]StreamST

type AvqueueMAP map[string]avQueue

type StreamST struct {
	Uuid         string
	Name         string
	Channels     ChannelMAP
	URL          string `json:"url" groups:"config"`
	Status       bool   `json:"status" groups:"config"`
	OnDemand     bool   `json:"on_demand" groups:"config"`
	DisableAudio bool   `json:"disable_audio" groups:"config"`
	Debug        bool   `json:"debug" groups:"config"`
	Codecs       []av.CodecData
	Cl           AvqueueMAP
	RunLock      bool
	msgStop      chan struct{}
}

type ChannelMAP map[string]ChannelST
type ChannelST struct {
	Name string
}

type avQueue struct {
	c chan av.Packet
}

func serveStreams() {
	gStreamListInfo.RunAllPersistStream()
	log.Println("serverStream Started")
}

// func (obj *StreamListInfoST) init(stream *StreamsMAP) {
// 	obj.Streams = stream
// }

func (obj *StreamListInfoST) RunAllPersistStream() {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	for iSuuid, tmpStream := range obj.Streams {
		if tmpStream.RunLock {
			continue
		}
		if tmpStream.OnDemand {
			continue
		}
		log.Println("RunStream :", iSuuid)

		tmpStream.RunLock = true
		tmpStream.msgStop = make(chan struct{})
		(obj.Streams)[iSuuid] = tmpStream
		go RTSPWorkerLoop(tmpStream.msgStop, iSuuid, tmpStream.URL, tmpStream.OnDemand, tmpStream.DisableAudio, tmpStream.Debug)
	}
}

/* func (obj *StreamListInfoST) RunIFNotRun(suuid string) {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	tmpStream, ok := (obj.Streams)[suuid]
	if !ok {
		return
	}

	if tmpStream.OnDemand && !tmpStream.RunLock {
		log.Println("RunIFNotRun", suuid)
		tmpStream.RunLock = true
		tmpStream.msgStop = make(chan struct{})
		(obj.Streams)[suuid] = tmpStream
		go RTSPWorkerLoop(tmpStream.msgStop, suuid, tmpStream.URL, tmpStream.OnDemand, tmpStream.DisableAudio, tmpStream.Debug)
	}
}*/

func (obj *StreamListInfoST) RunStream(suuid string) {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	tmpStream, ok := (obj.Streams)[suuid]
	if !ok || tmpStream.RunLock {
		return
	}
	log.Println("RunStream", suuid)
	tmpStream.RunLock = true
	tmpStream.msgStop = make(chan struct{})
	(obj.Streams)[suuid] = tmpStream
	go RTSPWorkerLoop(tmpStream.msgStop, suuid, tmpStream.URL, tmpStream.OnDemand, tmpStream.DisableAudio, tmpStream.Debug)
}

func (obj *StreamListInfoST) StopStream(suuid string) bool {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	tmpStream, ok := (obj.Streams)[suuid]
	if !ok {
		return false
	}
	if !tmpStream.RunLock {
		if tmpStream.OnDemand {
			return true
		}
		return false
	}

	log.Println("StopStream", suuid)
	close(tmpStream.msgStop)
	obj.mutex.Unlock()

	for {
		if !(obj.Streams)[suuid].RunLock {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	obj.mutex.Lock()
	return true
}

func (obj *StreamListInfoST) DeleteStream(suuid string) bool {
	if !obj.StopStream(suuid) {
		return false
	}
	obj.mutex.Lock() //??PYM_TEST_00000 how about done chan??
	defer obj.mutex.Unlock()

	delete((obj.Streams), suuid)
	return true
}

func (obj *StreamListInfoST) RunUnlock(suuid string) {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	tmpStream, ok := (obj.Streams)[suuid]
	if !ok {
		return
	}
	if tmpStream.RunLock {
		tmpStream.RunLock = false
		(obj.Streams)[suuid] = tmpStream
	}
}

func (obj *StreamListInfoST) HasViewer(suuid string) bool {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	if tmpStream, ok := (obj.Streams)[suuid]; ok && len(tmpStream.Cl) > 0 {
		return true
	}
	return false
}

func (obj *StreamListInfoST) cast(suuid string, pck av.Packet) {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	if _, ok := (obj.Streams)[suuid]; !ok {
		return
	}

	for _, iQueue := range (obj.Streams)[suuid].Cl {
		if len(iQueue.c) < cap(iQueue.c) {
			iQueue.c <- pck
		}
	}
}

func (obj *StreamListInfoST) ext(suuid string) bool {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	_, ok := (obj.Streams)[suuid]
	return ok
}

func (obj *StreamListInfoST) coAd(suuid string, codecs []av.CodecData) {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	if tmpStream, ok := (obj.Streams)[suuid]; ok {
		tmpStream.Codecs = codecs
		(obj.Streams)[suuid] = tmpStream
	}
}

func (obj *StreamListInfoST) coGe(suuid string) []av.CodecData {
	for i := 0; i < 100; i++ {
		obj.mutex.RLock()
		tmpStream, ok := (obj.Streams)[suuid]
		obj.mutex.RUnlock()
		if !ok {
			return nil
		}
		if tmpStream.Codecs != nil {
			//TODO Delete test
			for _, codec := range tmpStream.Codecs {
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
			return tmpStream.Codecs
		}
		time.Sleep(50 * time.Millisecond)
	}
	return nil
}

func (obj *StreamListInfoST) clAd(suuid string) (string, chan av.Packet) {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	cuuid := pseudoUUID()
	chAvQueue := make(chan av.Packet, 100)
	(obj.Streams)[suuid].Cl[cuuid] = avQueue{c: chAvQueue}
	return cuuid, chAvQueue
}

func (obj *StreamListInfoST) list() (string, []string) {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	var res []string
	var first string
	for iSuuid := range obj.Streams {
		if first == "" {
			first = iSuuid
		}
		res = append(res, iSuuid)
	}
	return first, res
}

func (obj *StreamListInfoST) list_url() []string {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	var res []string
	for iSuuid, tmpStream := range obj.Streams {

		res = append(res, iSuuid+", "+tmpStream.URL)
	}
	return res
}

func (obj *StreamListInfoST) clDe(suuid, cuuid string) {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	if _, ok := (obj.Streams)[suuid]; ok {
		delete((obj.Streams)[suuid].Cl, cuuid)
	}
}

func (obj *StreamListInfoST) apply_to_list(newStreamsList *StreamsMAP) bool {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()

	isListChanged := false
	var streamToDelete []string //group of iSuuid

	//same suuid -> check
	for iSuuid, oldStream := range obj.Streams {
		if newStream, ok := (*newStreamsList)[iSuuid]; ok {
			if oldStream.URL != newStream.URL { //different -> change
				oldStream.URL = newStream.URL
				(obj.Streams)[iSuuid] = oldStream
				if !isListChanged {
					isListChanged = true
				}
			}
			delete(*newStreamsList, iSuuid)
		} else {
			streamToDelete = append(streamToDelete, iSuuid)
		}
	}

	//no suuid ->delete
	for _, iSuuid := range streamToDelete {
		delete((obj.Streams), iSuuid)
		if !isListChanged {
			isListChanged = true
		}
	}

	//new suuid ->add
	for iSuuid, newStream := range *newStreamsList {
		(obj.Streams)[iSuuid] = newStream
		if !isListChanged {
			isListChanged = true
		}
	}

	return isListChanged

}

func RTSPWorkerLoop(msgStop <-chan struct{}, suuid, url string, OnDemand, DisableAudio, Debug bool) {
	defer gStreamListInfo.RunUnlock(suuid)
	for {
		log.Println("Stream Try Connect", suuid)
		err := RTSPWorker(msgStop, suuid, url, OnDemand, DisableAudio, Debug)
		if err != nil {
			log.Println(err)
			gConfig.LastError = err
			if err == ErrorStreamExitStopMsgReceived {
				return
			}
		}
		if OnDemand && !gStreamListInfo.HasViewer(suuid) {
			log.Println(ErrorStreamExitNoViewer)
			return
		}
		time.Sleep(1 * time.Second)
	}
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

/////

// type StreamListST struct {
// 	mutex         sync.RWMutex
// 	Streams       StreamsMAP `json:"streams" groups:"config"`
// 	Streams_extra StreamsMAP `json:"streams_extra" groups:"config"`
// }

const StreamListJsonFile = "stream_list.json"

func (obj *StreamListInfoST) loadList() {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	data, err := os.ReadFile(StreamListJsonFile)
	if err == nil {
		err = json.Unmarshal(data, &obj)
		if err != nil {
			log.Fatalln(err)
		}
		for iUuid, tmpStream := range obj.Streams {
			tmpStream.Channels = make(ChannelMAP)
			tmpStream.Cl = make(AvqueueMAP)
			tmpStream.Uuid = iUuid
			tmpStream.Name = iUuid
			tmpStream.Channels["0"] = ChannelST{""}
			tmpStream.RunLock = false
			//tmpStream.msgStop = make(chan struct{})
			obj.Streams[iUuid] = tmpStream
		}
	} else {
		obj.Streams = make(StreamsMAP)
	}

}

func (obj *StreamListInfoST) SaveList() error {
	// log.WithFields(logrus.Fields{
	// 	"module": "stream_list",
	// 	"func":   "NewStreamCore",
	// }).Debugln("Saving configuration to", StreamListJsonFile)
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

	err = os.WriteFile(StreamListJsonFile, JsonData, 0644)
	if err != nil {
		// log.WithFields(logrus.Fields{
		// 	"module": "stream_list",
		// 	"func":   "SaveList",
		// 	"call":   "WriteFile",
		// }).Errorln(err.Error())
		return err
	}

	return nil
}
