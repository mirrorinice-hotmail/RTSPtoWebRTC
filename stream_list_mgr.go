package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/deepch/vdk/codec/h264parser"
	"github.com/hashicorp/go-version"
	"github.com/liip/sheriff"

	"github.com/deepch/vdk/av"
)

/*"github.com/imdario/mergo"*/

func serveStreams() {
	gStreamListInfo.RunAllPersistStream()
	log.Println("serverStream Started")
}

// /////////////////////////////////////////////////////////////////////////////
var gStreamListInfo = StreamListInfoST{
	Streams:       make(StreamsMAP),
	Streams_extra: make(StreamsMAP),
	pseudoUUID: func() (uuid string) {
		b := make([]byte, 16)
		_, err := rand.Read(b)
		if err != nil {
			fmt.Println("Error: ", err)
			return
		}
		uuid = fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
		return
	},
}

type StreamListInfoST struct {
	mutex         sync.RWMutex
	Streams       StreamsMAP `json:"streams" groups:"config"`
	Streams_extra StreamsMAP `json:"streams_extra" groups:"config"`
	//LastError error
	pseudoUUID func() (uuid string)
}

type StreamsMAP map[string]StreamST
type StreamST struct {
	Uuid         string
	CctvName     string `json:"cctv_name" groups:"config"`
	CctvIp       string `json:"cctv_ip" groups:"config"`
	Channels     ChannelMAP
	RtspUrl      string `json:"url" groups:"config"`
	Status       bool   `json:"status" groups:"config"`
	OnDemand     bool   `json:"on_demand" groups:"config"`
	DisableAudio bool   `json:"disable_audio" groups:"config"`
	Debug        bool   `json:"debug" groups:"config"`
	Codecs       []av.CodecData
	avQue        AvqueueMAP
	RunLock      bool
	msgStop      chan struct{}
}

func (obj *StreamST) WorkerLoop() {
	defer gStreamListInfo.RunUnlock(obj.Uuid)
	//sleepCount := 1
	for {
		sleepTime := 1 * time.Second
		log.Println("Stream Connect : '", obj.Uuid, "'")
		err := RTSPWorker(obj.msgStop, obj.Uuid, obj.RtspUrl, obj.OnDemand, obj.DisableAudio, obj.Debug)
		if err != nil {
			gConfig.LastError = err
			strErr := err.Error()
			log.Println("Stream Err : '", obj.Uuid, "' -", strErr)
			if strings.Contains(strErr, "dial tcp") && strings.Contains(strErr, "i/o timeout") {
				sleepTime = 10 * time.Second
			} else {
				sleepTime = 3 * time.Second
			}

			if err == ErrorStreamExit_StopMsgReceived {
				return
			}
		}
		if obj.OnDemand && !gStreamListInfo.HasViewer(obj.Uuid) {
			log.Println(ErrorStreamExit_NoViewer)
			return
		}

		select {
		case <-obj.msgStop:
			log.Println("WorkerLoop breaked: msg 'stop'")
			return
		case <-time.After(sleepTime):
		}
		//time.Sleep(sleepTime)
	}
}

type ChannelMAP map[string]ChannelST
type ChannelST struct {
	Name string
}

type AvqueueMAP map[string]avQueue
type avQueue struct {
	c chan av.Packet
}

// ////////////////////////////////////////////////////////////////////
func (obj *StreamListInfoST) RunAllPersistStream() {
	for iSuuid, tmpStream := range obj.Streams {
		if tmpStream.OnDemand {
			continue
		}
		obj.RunStream(iSuuid)
	}
}

func (obj *StreamListInfoST) StopAllStream() {
	for iSuuid, tmpStream := range obj.Streams {
		if tmpStream.OnDemand {
			continue
		}
		obj.StopStream(iSuuid)
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
	go tmpStream.WorkerLoop() //RTSPWorkerLoop(tmpStream...)
}

func (obj *StreamListInfoST) StopStream(suuid string) bool {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	tmpStream, ok := (obj.Streams)[suuid]
	if !ok {
		return false
	}
	if !tmpStream.RunLock {
		return tmpStream.OnDemand
	}

	log.Println("Stream Stop :'", suuid, "'")
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

func (obj *StreamListInfoST) SaveStream(saveParam *streamSaveParamST) bool {
	// {Uuid : saveParam.Suuid, CctvName : saveParam.Name, RtspUrl : saveParam.Url,
	// OnDemand : saveParam.OnDemand, Debug: saveParam.Debug }
	Suuid := saveParam.Suuid
	if obj.exist(Suuid) { //edit/change parameter
		if !obj.StopStream(Suuid) {
			return false
		}
		obj.mutex.Lock()
		tmpStream := obj.Streams[Suuid]
		tmpStream.CctvName = saveParam.Name
		tmpStream.RtspUrl = saveParam.Url
		tmpStream.OnDemand = saveParam.OnDemand
		tmpStream.Debug = saveParam.Debug
		obj.Streams[Suuid] = tmpStream
		obj.mutex.Unlock()

		obj.RunStream(Suuid)
		return true

	} else { //add parameter
		obj.mutex.Lock()
		tmpStream := StreamST{
			Uuid:     Suuid,
			CctvName: saveParam.Name,
			//CctvIp:       "",
			Channels: make(ChannelMAP),
			RtspUrl:  saveParam.Url,
			//Status:       false,
			OnDemand:     saveParam.OnDemand,
			DisableAudio: true,
			Debug:        saveParam.Debug,
			Codecs:       nil,
			avQue:        make(AvqueueMAP),
			RunLock:      false,
		}
		tmpStream.Channels["0"] = ChannelST{}
		obj.Streams[Suuid] = tmpStream
		obj.mutex.Unlock()

		obj.RunStream(Suuid)
		return true
	}

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
	log.Println("Unlock run :'", suuid, "'")
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
	obj.mutex.RLock()
	defer obj.mutex.RUnlock()
	if tmpStream, ok := (obj.Streams)[suuid]; ok && len(tmpStream.avQue) > 0 {
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

	for _, iQueue := range (obj.Streams)[suuid].avQue {
		if len(iQueue.c) < cap(iQueue.c) {
			iQueue.c <- pck
		}
	}
}

func (obj *StreamListInfoST) exist(suuid string) bool {
	obj.mutex.RLock()
	defer obj.mutex.RUnlock()
	_, ok := (obj.Streams)[suuid]
	return ok
}

func (obj *StreamListInfoST) setCodec(suuid string, codecs []av.CodecData) {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	if tmpStream, ok := (obj.Streams)[suuid]; ok {
		tmpStream.Codecs = codecs
		(obj.Streams)[suuid] = tmpStream
	}
}

func (obj *StreamListInfoST) getCodec(suuid string) []av.CodecData {
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

func (obj *StreamListInfoST) addAvque(suuid string) (string, chan av.Packet) {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	cuuid := obj.pseudoUUID()
	chAvQueue := make(chan av.Packet, 100)
	(obj.Streams)[suuid].avQue[cuuid] = avQueue{c: chAvQueue}
	return cuuid, chAvQueue
}

func (obj *StreamListInfoST) delAvque(suuid, cuuid string) {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	if _, ok := (obj.Streams)[suuid]; ok {
		delete((obj.Streams)[suuid].avQue, cuuid)
	}
}

func (obj *StreamListInfoST) GetFirstStreamUuid() (uuid string, result bool) {
	obj.mutex.RLock()
	defer obj.mutex.RUnlock()
	for iSuuid := range obj.Streams {
		return iSuuid, true
	}
	return "", false
}

/*
func (obj *StreamListInfoST) list_url() []string {
	obj.mutex.RLock()
	defer obj.mutex.RUnlock()
	var res []string
	for iSuuid, tmpStream := range obj.Streams {
		res = append(res, iSuuid+", "+tmpStream.URL)
	}
	return res
}*/

func (obj *StreamListInfoST) apply_to_list(newStreamsList StreamsMAP) bool {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()

	isListChanged := false
	var streamToDelete []string //group of iSuuid

	//same suuid -> check
	for iSuuid, oldStream := range obj.Streams {
		if newStream, ok := (newStreamsList)[iSuuid]; ok {
			change_found := false
			if oldStream.RtspUrl != newStream.RtspUrl { //different -> change
				change_found = true
				oldStream.RtspUrl = newStream.RtspUrl
			}
			if oldStream.CctvName != newStream.CctvName { //different -> change
				change_found = true
				oldStream.CctvName = newStream.CctvName
			}

			if change_found {
				(obj.Streams)[iSuuid] = oldStream
				if !isListChanged {
					isListChanged = true
				}
			}
			delete(newStreamsList, iSuuid)
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
	for iSuuid, newStream := range newStreamsList {
		(obj.Streams)[iSuuid] = newStream
		if !isListChanged {
			isListChanged = true
		}
	}

	return isListChanged

}

/*func RTSPWorkerLoop(msgStop <-chan struct{}, suuid, url string, OnDemand, DisableAudio, Debug bool) {
	defer gStreamListInfo.RunUnlock(suuid)
	//sleepCount := 1
	for {
		sleepTime := 1 * time.Second
		log.Println("Stream Connect : '", suuid, "'")
		err := RTSPWorker(msgStop, suuid, url, OnDemand, DisableAudio, Debug)
		if err != nil {
			gConfig.LastError = err
			strErr := err.Error()
			log.Println("Stream Err : '", suuid, "' -", strErr)
			if strings.Contains(strErr, "dial tcp") && strings.Contains(strErr, "i/o timeout") {
				sleepTime = 10 * time.Second
			} else {
				sleepTime = 3 * time.Second
			}

			if err == ErrorStreamExit_StopMsgReceived {
				return
			}
		}
		if OnDemand && !gStreamListInfo.HasViewer(suuid) {
			log.Println(ErrorStreamExit_NoViewer)
			return
		}

		select {
		case <-msgStop:
			err = ErrorStreamExit_StopMsgReceived
			log.Println("RTSPWorkerLoop err: ", err)
			return
		case <-time.After(sleepTime):
		}
		//time.Sleep(sleepTime)
	}
}*/

///////////////////////////////////////////

const StreamListJsonFile = "stream_list.json"

func (obj *StreamListInfoST) loadList() {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	data, err := os.ReadFile(StreamListJsonFile)
	if err == nil {
		err = json.Unmarshal(data, &obj)
		if err != nil {
			log.Fatalln(err)
			return
		}
		for iUuid, tmpStream := range obj.Streams {
			tmpStream.Channels = make(ChannelMAP)
			tmpStream.avQue = make(AvqueueMAP)
			tmpStream.Uuid = iUuid
			tmpStream.Channels["0"] = ChannelST{""}
			tmpStream.RunLock = false
			obj.Streams[iUuid] = tmpStream
		}

		if obj.Streams == nil {
			obj.Streams = make(StreamsMAP)
		}
	}
}

func (obj *StreamListInfoST) SaveList() error {
	obj.mutex.RLock()
	defer obj.mutex.RUnlock()
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
