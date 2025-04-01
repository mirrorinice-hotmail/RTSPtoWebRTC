package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/deepch/vdk/av"

	webrtc "github.com/deepch/vdk/format/webrtcv3"
	"github.com/gin-gonic/gin"
)

/////////////////////////////////////////////////////////
// web rtc signalling apis

// stream codec
func HTTPAPIServerStreamCodec(c *gin.Context) {
	strSuuid := c.Param("uuid")
	if !gStreamListInfo.exist(strSuuid) {
		log.Println("HTTPAPIServerStreamCodec error: unknown id")
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown id"})
		return
	}
	gStreamListInfo.RunStream(strSuuid) //gConfig.RunIFNotRun(strSuuid)
	codecs := gStreamListInfo.getCodec2(strSuuid)
	if codecs == nil {
		return
	}
	var tmpCodec []JCodec
	for _, codec := range codecs {
		if codec.Type() != av.H264 && codec.Type() != av.PCM_ALAW && codec.Type() != av.PCM_MULAW && codec.Type() != av.OPUS {
			log.Println("Codec Not Supported WebRTC ignore this track", codec.Type())
			continue
		}
		if codec.Type().IsVideo() {
			tmpCodec = append(tmpCodec, JCodec{Type: "video"})
		} else {
			tmpCodec = append(tmpCodec, JCodec{Type: "audio"})
		}
	}
	b, err := json.Marshal(tmpCodec)
	if err == nil {
		_, err = c.Writer.Write(b)
		if err != nil {
			log.Println("Write Codec Info error", err)
			return
		}
	}
}

// stream video over WebRTC
func HTTPAPIServerStreamWebRTC(c *gin.Context) {
	strSuuid := c.PostForm("suuid")
	if !gStreamListInfo.exist(strSuuid) {
		log.Println("HTTPAPIServerStreamWebRTC error: unknown id")
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown id"})
		return
	}
	gStreamListInfo.RunStream(strSuuid) //gConfig.RunIFNotRun(strSuuid)
	codecs := gStreamListInfo.getCodec2(strSuuid)
	if codecs == nil {
		log.Println("Stream Codec Not Found")
		return
	}
	var AudioOnly bool
	if len(codecs) == 1 && codecs[0].Type().IsAudio() {
		AudioOnly = true
	}
	muxerWebRTC := webrtc.NewMuxer(
		webrtc.Options{
			ICEServers:    gConfig.GetICEServers(),
			ICEUsername:   gConfig.GetICEUsername(),
			ICECredential: gConfig.GetICECredential(),
			PortMin:       gConfig.GetWebRTCPortMin(),
			PortMax:       gConfig.GetWebRTCPortMax(),
		},
	)
	answer, err := muxerWebRTC.WriteHeader(codecs, c.PostForm("data"))
	if err != nil {
		log.Println("WriteHeader", err)
		return
	}
	_, err = c.Writer.Write([]byte(answer))
	if err != nil {
		log.Println("Write", err)
		return
	}
	go func() {
		//strSuuid := c.PostForm("suuid")
		cid, ch := gStreamListInfo.addAvque(strSuuid)
		defer gStreamListInfo.delAvque(strSuuid, cid)
		defer muxerWebRTC.Close()
		var videoStart bool
		noVideo := time.NewTimer(10 * time.Second)
		for {
			select {
			case <-noVideo.C:
				log.Println("noVideo")
				return
			case pck := <-ch:
				if pck.IsKeyFrame || AudioOnly {
					noVideo.Reset(10 * time.Second)
					videoStart = true
				}
				if !videoStart && !AudioOnly {
					continue
				}
				err = muxerWebRTC.WritePacket(pck)
				if err != nil {
					log.Println("WritePacket", err)
					return
				}
			}
		}
	}()
}
