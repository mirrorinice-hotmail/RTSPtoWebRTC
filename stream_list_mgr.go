package main

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/deepch/vdk/codec/h264parser"

	"github.com/deepch/vdk/av"
)

/*"github.com/imdario/mergo"*/

var gStreamListInfo StreamListInfoST

// ///////////////////////////////////
type StreamListInfoST struct {
	mutex   sync.RWMutex
	Streams *map[string]StreamST `json:"streams" groups:"config"`
	//LastError error
}

// StreamST struct
type StreamST struct {
	Uuid         string
	URL          string `json:"url" groups:"config"`
	Status       bool   `json:"status" groups:"config"`
	OnDemand     bool   `json:"on_demand" groups:"config"`
	DisableAudio bool   `json:"disable_audio" groups:"config"`
	Debug        bool   `json:"debug" groups:"config"`
	RunLock      bool
	Codecs       []av.CodecData
	Cl           map[string]avQueue
}

type avQueue struct {
	c chan av.Packet
}

func serveStreams() {
	gStreamListInfo.RunAllStream()
	log.Println("serverStream Started")
}

func (obj *StreamListInfoST) RunAllStream() {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	for iSuuid, tmpStream := range *obj.Streams {
		if tmpStream.RunLock {
			continue
		}
		log.Println("RunStream of all", iSuuid)
		tmpStream.RunLock = true
		(*obj.Streams)[iSuuid] = tmpStream
		//*
		go RTSPWorkerLoop(iSuuid, tmpStream.URL, tmpStream.OnDemand, tmpStream.DisableAudio, tmpStream.Debug)
		/*/
		go obj.StreamWorkerLoop(iSuuid, tmpStream)
		// */
	}
}

func (obj *StreamListInfoST) RunIFNotRun(suuid string) {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	if tmpStream, ok := (*obj.Streams)[suuid]; ok {
		if tmpStream.OnDemand && !tmpStream.RunLock {
			log.Println("RunIFNotRun", suuid)
			tmpStream.RunLock = true
			(*obj.Streams)[suuid] = tmpStream
			//*
			go RTSPWorkerLoop(suuid, tmpStream.URL, tmpStream.OnDemand, tmpStream.DisableAudio, tmpStream.Debug)
			/*/
			go obj.StreamWorkerLoop(suuid, tmpStream)
			// */
		}
	}
}

func (obj *StreamListInfoST) RunStream(suuid string) {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	tmpStream, ok := (*obj.Streams)[suuid]
	if !ok || tmpStream.RunLock {
		return
	}
	log.Println("RunStream", suuid)
	tmpStream.RunLock = true
	(*obj.Streams)[suuid] = tmpStream
	//*
	go RTSPWorkerLoop(suuid, tmpStream.URL, tmpStream.OnDemand, tmpStream.DisableAudio, tmpStream.Debug)
	/*/
	go obj.StreamWorkerLoop(suuid, stream)
	// */

}

func (obj *StreamListInfoST) Runlock(suuid string, onOff bool) {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	tmpStream, ok := (*obj.Streams)[suuid]
	if !ok || tmpStream.RunLock {
		return
	}

	if tmpStream.RunLock {
		tmpStream.RunLock = onOff
		(*obj.Streams)[suuid] = tmpStream
	}

}

func (obj *StreamListInfoST) HasViewer(suuid string) bool {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	if tmpStream, ok := (*obj.Streams)[suuid]; ok && len(tmpStream.Cl) > 0 {
		return true
	}
	return false
}

func (obj *StreamListInfoST) cast(suuid string, pck av.Packet) {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	if _, ok := (*obj.Streams)[suuid]; !ok {
		return
	}

	for _, iQueue := range (*obj.Streams)[suuid].Cl {
		if len(iQueue.c) < cap(iQueue.c) {
			iQueue.c <- pck
		}
	}
}

func (obj *StreamListInfoST) ext(suuid string) bool {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	_, ok := (*obj.Streams)[suuid]
	return ok
}

func (obj *StreamListInfoST) coAd(suuid string, codecs []av.CodecData) {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	if tmpStream, ok := (*obj.Streams)[suuid]; ok {
		tmpStream.Codecs = codecs
		(*obj.Streams)[suuid] = tmpStream
	}
}

func (obj *StreamListInfoST) coGe(suuid string) []av.CodecData {
	for i := 0; i < 100; i++ {
		obj.mutex.RLock()
		tmpStream, ok := (*obj.Streams)[suuid]
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
	(*obj.Streams)[suuid].Cl[cuuid] = avQueue{c: chAvQueue}
	return cuuid, chAvQueue
}

func (obj *StreamListInfoST) list() (string, []string) {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	var res []string
	var first string
	for iSuuid := range *obj.Streams {
		if first == "" {
			first = iSuuid
		}
		res = append(res, iSuuid)
	}
	return first, res
}
func (obj *StreamListInfoST) clDe(suuid, cuuid string) {
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	if _, ok := (*obj.Streams)[suuid]; ok {
		delete((*obj.Streams)[suuid].Cl, cuuid)
	}
}

func (obj *StreamListInfoST) update_list(rows *sql.Rows) {
	//gStreamListInfo.
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	for rows.Next() {
		var val_stream_id, val_rtsp_01, val_rtsp_02, val_cctv_nm string
		err := rows.Scan(&val_stream_id, &val_rtsp_01, &val_cctv_nm)
		if err != nil {
			panic(err)
		}
		fmt.Printf("stream list: stream_id(%s), rtsp_01(%s), rtsp_02(%s) , cctv_nm(%s)\n",
			val_stream_id, val_rtsp_01, val_rtsp_02, val_cctv_nm)

		var tmpStream StreamST
		if tmpStream, ok := (*gStreamListInfo.Streams)[val_stream_id]; ok {
			tmpStream.URL = val_rtsp_01
		} else {
			tmpStream = StreamST{
				Uuid:         val_stream_id,
				URL:          val_rtsp_01,
				Status:       false,
				OnDemand:     false,
				DisableAudio: true,
				Debug:        false,
				RunLock:      false,
				Codecs:       nil,
				Cl:           make(map[string]avQueue)}
		}
		(*gStreamListInfo.Streams)[val_stream_id] = tmpStream
	}

	gConfig.SaveConfig() //??PYM_TEST_00000
}

func RTSPWorkerLoop(suuid, url string, OnDemand, DisableAudio, Debug bool) {
	defer gStreamListInfo.Runlock(suuid, false)
	for {
		log.Println("Stream Try Connect", suuid)
		err := RTSPWorker(suuid, url, OnDemand, DisableAudio, Debug)
		if err != nil {
			log.Println(err)
			gConfig.LastError = err
		}
		if OnDemand && !gStreamListInfo.HasViewer(suuid) {
			log.Println(ErrorStreamExitNoViewer)
			return
		}
		time.Sleep(1 * time.Second)
	}
}

func (obj *StreamListInfoST) StreamWorkerLoop(suuid string, stream StreamST) {
	defer obj.Runlock(suuid, false)
	for {
		log.Println("Stream Try Connect", suuid)
		err := RTSPWorker(suuid, stream.URL, stream.OnDemand, stream.DisableAudio, stream.Debug)
		if err != nil {
			log.Println(err)
			gConfig.LastError = err
		}
		if stream.OnDemand && !obj.HasViewer(suuid) {
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
