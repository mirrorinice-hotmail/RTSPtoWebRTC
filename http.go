package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/deepch/vdk/av"

	webrtc "github.com/deepch/vdk/format/webrtcv3"
	"github.com/gin-gonic/gin"
)

type JCodec struct {
	Type string
}

var serverHttp *http.Server

func serveHTTP() {
	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()
	router.Use(CORSMiddleware())

	if _, err := os.Stat("./web"); !os.IsNotExist(err) {
		router.LoadHTMLGlob("web/templates/*")
		router.GET("/", HTTPAPIStreamList)
		router.GET("/stream/list", HTTPAPIStreamList)
		router.GET("/stream/edit", HTTPAPIStreamEdit)
		router.GET("/stream/edit/:uuid", HTTPAPIStreamEdit)
		router.GET("/stream/add", HTTPAPIStreamAdd)
		router.POST("/stream/save", HTTPAPIStreamSave)
		router.GET("/stream/delete/:uuid", HTTPAPIStreamDelete)
		router.GET("/stream/player", HTTPAPIServerStreamPlayer)
		router.GET("/stream/player/:uuid", HTTPAPIServerStreamPlayer)
		router.GET("/stream/updatelist", HTTPAPIServerStreamUpdateList)
	}
	router.GET("/stream/codec/:uuid", HTTPAPIServerStreamCodec)
	router.POST("/stream/receiver/:uuid", HTTPAPIServerStreamWebRTC)

	router.StaticFS("/static", http.Dir("web/static"))

	log.Println("ServerHTTP start")
	//*
	serverHttp = &http.Server{
		Addr:    gConfig.HttpServer.HTTPPort,
		Handler: router,
	}
	log.Println("ServerHTTP ListenAndServe")
	err := serverHttp.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("ServerHTTP listen: %s\n", err)
	}
	/*/
	log.Println("ServerHTTP router run")
	err := router.Run(gConfig.HttpServer.HTTPPort)
	if err != nil {
		log.Fatalln("Start HTTP Server error", err)
	}
	// */
	log.Println("ServerHTTP stopped")

}

// index
func HTTPAPIServerIndex(c *gin.Context) {
	_, all := gStreamListInfo.list()
	if len(all) > 0 {
		c.Header("Cache-Control", "no-cache, max-age=0, must-revalidate, no-store")
		c.Header("Access-Control-Allow-Origin", "*")
		c.Redirect(http.StatusMovedPermanently, "stream/player/"+all[0])
	} else {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"port":    gConfig.HttpServer.HTTPPort,
			"version": time.Now().String(),
		})
	}
}

// list
func HTTPAPIStreamList(c *gin.Context) {
	_, all := gStreamListInfo.list()
	if len(all) > 0 {
		c.Header("Cache-Control", "no-cache, max-age=0, must-revalidate, no-store")
		c.Header("Access-Control-Allow-Origin", "*")
	}

	media_svr_addr := gConfig.HttpServer.HTTPHost + gConfig.HttpServer.HTTPPort
	log.Println("HTTPAPIStreamList() media_svr_addr: " + media_svr_addr)
	pagename := "stream_list"
	c.HTML(http.StatusOK, pagename+".html", gin.H{
		"media_svr_addr": media_svr_addr,
		"port":           gConfig.HttpServer.HTTPPort,
		"streams":        gStreamListInfo.Streams,
		"version":        time.Now().String(),
		"page":           pagename,
	})
}

// edit
func HTTPAPIStreamEdit(c *gin.Context) {
	strSuuid := c.Param("uuid")
	if !gStreamListInfo.ext(strSuuid) {
		log.Println("Stream Not Found")
		c.HTML(http.StatusOK, "index.html", gin.H{
			"port":    gConfig.HttpServer.HTTPPort,
			"version": time.Now().String(),
		})
		return
	}

	pagename := "edit_stream"
	streamsJSON, _ := json.Marshal(gStreamListInfo.Streams)
	c.HTML(http.StatusOK, pagename+".html", gin.H{
		"port":       gConfig.HttpServer.HTTPPort,
		"streamJson": string(streamsJSON),
		"streams":    gStreamListInfo.Streams,
		"streamone":  (gStreamListInfo.Streams)[strSuuid],
		"uuid":       strSuuid,
		"version":    time.Now().String(),
		"page":       "Edit Stream",
	})

}

// add
func HTTPAPIStreamAdd(c *gin.Context) {
	pagename := "edit_stream"
	streamsJSON, _ := json.Marshal(gStreamListInfo.Streams)
	c.HTML(http.StatusOK, pagename+".html", gin.H{
		"port":       gConfig.HttpServer.HTTPPort,
		"streamJson": string(streamsJSON),
		"streams":    gStreamListInfo.Streams,
		"streamone":  StreamST{Uuid: "", Name: "", URL: ""},
		"uuid":       "",
		"version":    time.Now().String(),
		"page":       "Add Stream",
	})
}

// delete
func HTTPAPIStreamDelete(c *gin.Context) {
	strSuuid := c.Param("uuid")
	if !gStreamListInfo.ext(strSuuid) {
		log.Println("Stream Not Found")
		c.HTML(http.StatusOK, "index.html", gin.H{
			"port":    gConfig.HttpServer.HTTPPort,
			"version": time.Now().String(),
		})
		return
	}

	if !gStreamListInfo.DeleteStream(strSuuid) {
		return
	}

	gStreamListInfo.SaveList()
	HTTPAPIStreamList(c)
}

// stream player
func HTTPAPIServerStreamPlayer(c *gin.Context) {
	strSuuid := c.Param("uuid")
	if !gStreamListInfo.ext(strSuuid) {
		log.Println("Stream Not Found")
		c.HTML(http.StatusOK, "index.html", gin.H{
			"port":    gConfig.HttpServer.HTTPPort,
			"version": time.Now().String(),
		})
		return
	}

	media_svr_addr := gConfig.HttpServer.HTTPHost + gConfig.HttpServer.HTTPPort
	log.Println("HTTPAPIStreamList() media_svr_addr: " + media_svr_addr)

	c.HTML(http.StatusOK, "player.html", gin.H{
		"media_svr_addr": media_svr_addr,
		"port":           gConfig.HttpServer.HTTPPort,
		"suuid":          strSuuid,
		"version":        time.Now().String(),
	})
}

func HTTPAPIServerStreamUpdateList(c *gin.Context) {
	//gCctvListMgr.request_updatelist()
	log.Println("HTTPAPIServerStreamUpdateList: started")
	if gCctvListMgr.updateList() {
		log.Println("HTTPAPIServerStreamUpdateList: sucess")
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	} else {
		log.Println("HTTPAPIServerStreamUpdateList: failure")
		c.JSON(http.StatusFailedDependency, gin.H{"status": "failure"})
	}

	log.Println("HTTPAPIServerStreamUpdateList: end")
}

// stream codec
func HTTPAPIServerStreamCodec(c *gin.Context) {
	strSuuid := c.Param("uuid")
	if !gStreamListInfo.ext(strSuuid) {
		log.Println("Stream Not Found")
		c.HTML(http.StatusOK, "index.html", gin.H{
			"port":    gConfig.HttpServer.HTTPPort,
			"version": time.Now().String(),
		})
		return
	}
	gStreamListInfo.RunStream(strSuuid) //gConfig.RunIFNotRun(strSuuid)
	codecs := gStreamListInfo.coGe(strSuuid)
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

type streamSaveParamST struct {
	Suuid    string `json:"suuid" binding:"required"`
	Url      string `json:"url" binding:"required"`
	Debug    bool   `json:"debug"`
	OnDemand bool   `json:"on_demand"`
}

// save stream info
func HTTPAPIStreamSave(c *gin.Context) {
	log.Println("HTTPAPIStreamSave start...")
	authHeader := c.GetHeader("Authorization")
	expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("demo:demo"))

	if authHeader != expectedAuth {
		log.Println("HTTPAPIStreamSave error: Unau thorized...")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var saveParam streamSaveParamST
	if err := c.ShouldBindJSON(&saveParam); err != nil {
		log.Println("HTTPAPIStreamSave error: Invalid JSON format...")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}

	log.Printf("Received: suuid(%s) url(%s) debug (%t) ondemand (%t)\n",
		saveParam.Suuid,
		saveParam.Url,
		saveParam.Debug,
		saveParam.OnDemand)

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// stream video over WebRTC
func HTTPAPIServerStreamWebRTC(c *gin.Context) {
	strSuuid := c.PostForm("suuid")
	if !gStreamListInfo.ext(strSuuid) {
		log.Println("Stream Not Found")
		c.HTML(http.StatusOK, "index.html", gin.H{
			"port":    gConfig.HttpServer.HTTPPort,
			"version": time.Now().String(),
		})
		return
	}
	gStreamListInfo.RunStream(strSuuid) //gConfig.RunIFNotRun(strSuuid)
	codecs := gStreamListInfo.coGe(strSuuid)
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
		strSuuid := c.PostForm("suuid")
		cid, ch := gStreamListInfo.clAd(strSuuid)
		defer gStreamListInfo.clDe(strSuuid, cid)
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

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization, x-access-token")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

type ResponseError struct {
	Error string `json:"error"`
}
