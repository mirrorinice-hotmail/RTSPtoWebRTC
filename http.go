package main

import (
	"encoding/base64"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

type JCodec struct {
	Type string
}

const (
	PAGE_STREAM_LIST = "stream list"
	//PAGE_STREAM_EDIT = "stream editing"
	//PAGE_STREAM_ADD  = "stream adding"
)

var pageNameMAP = map[string]string{
	PAGE_STREAM_LIST: "stream_list.html",
	//PAGE_STREAM_EDIT: "edit_stream.html",
	//PAGE_STREAM_ADD:  "edit_stream.html",
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
		router.POST("/stream/save", HTTPAPIStreamSave)
		router.GET("/stream/delete/:uuid", HTTPAPIStreamDelete)
		router.GET("/stream/player", HTTPAPIServerStreamPlayer)
		router.GET("/stream/player/:uuid", HTTPAPIServerStreamPlayer)
		router.GET("/stream/updatelist", HTTPAPIServerStreamUpdateList)
		router.POST("/stream/uploadlist", HTTPAPIServerStreamUploadList)
		router.StaticFS("/static", http.Dir("web/static"))

	}
	//Webrtc Signalling
	router.GET("/stream/codec/:uuid", HTTPAPIServerStreamCodec)
	router.POST("/stream/receiver/:uuid", HTTPAPIServerStreamWebRTC)

	//service starting
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

// list
func HTTPAPIStreamList(c *gin.Context) {
	_, ok := gStreamListInfo.GetFirstStreamUuid()
	if ok {
		c.Header("Cache-Control", "no-cache, max-age=0, must-revalidate, no-store")
		c.Header("Access-Control-Allow-Origin", "*")
	}

	media_svr_addr := gConfig.HttpServer.HTTPHost + gConfig.HttpServer.HTTPPort
	log.Println("HTTPAPIStreamList() media_svr_addr: " + media_svr_addr)

	c.HTML(http.StatusOK, pageNameMAP[PAGE_STREAM_LIST], gin.H{
		"media_svr_addr": media_svr_addr,
		"port":           gConfig.HttpServer.HTTPPort,
		"streams":        gStreamListInfo.Streams,
		"version":        time.Now().String(),
		"page":           PAGE_STREAM_LIST,
	})
}

// delete
func HTTPAPIStreamDelete(c *gin.Context) {
	log.Println("HTTPAPIStreamDelete: started")
	authHeader := c.GetHeader("Authorization")
	expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("rino:ese"))

	if authHeader != expectedAuth {
		log.Println("HTTPAPIStreamDelete error: Unau thorized...")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	strSuuid := c.Param("uuid")
	if !gStreamListInfo.exist(strSuuid) {
		log.Println("HTTPAPIStreamDelete error: unknown id")
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown id"})
		return
	}

	if gStreamListInfo.DeleteStream(strSuuid) {
		gStreamListInfo.SaveList()
		log.Println("HTTPAPIStreamDelete: success")
		c.JSON(http.StatusOK, gin.H{"status": "success"})

	} else {
		log.Println("HTTPAPIStreamDelete: failure")
		c.JSON(http.StatusFailedDependency, gin.H{"status": "failure"})

	}
	log.Println("HTTPAPIStreamDelete: end")
}

// stream player
func HTTPAPIServerStreamPlayer(c *gin.Context) {
	strSuuid := c.Param("uuid")
	if !gStreamListInfo.exist(strSuuid) {
		log.Println("HTTPAPIServerStreamPlayer error: unknown id")
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown id"})
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
	log.Println("HTTPAPIServerStreamUpdateList: started")
	authHeader := c.GetHeader("Authorization")
	expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("rino:ese"))

	if authHeader != expectedAuth {
		log.Println("HTTPAPIServerStreamUpdateList error: Unau thorized...")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	/*
		if gCctvListMgr.db_add_samples() {
			log.Println("HTTPAPIServerStreamUpdateList: sucess")
			c.JSON(http.StatusOK, gin.H{"status": "success"})
		} else {
			log.Println("HTTPAPIServerStreamUpdateList: failure")
			c.JSON(http.StatusFailedDependency, gin.H{"status": "failure"})
		}
		/*/
	newStreams := gCctvListMgr.update_stream_list()
	if newStreams != nil {
		gStreamListInfo.StopAllStream()
		isListChanged := gStreamListInfo.apply_to_list(newStreams)
		if isListChanged {
			gStreamListInfo.SaveList()
		}
		gStreamListInfo.RunAllPersistStream()
		log.Println("HTTPAPIServerStreamUpdateList: sucess")
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	} else {
		log.Println("HTTPAPIServerStreamUpdateList: failure")
		c.JSON(http.StatusFailedDependency, gin.H{"status": "failure"})
	}
	// */

	log.Println("HTTPAPIServerStreamUpdateList: end")
}

type streamSaveParamST struct {
	NewStream bool   `json:"new_stream"`
	Suuid     string `json:"suuid" binding:"required"`
	Name      string `json:"name" binding:"required"`
	RtspUrl   string `json:"url" binding:"required"`
	RtspUrl_2 string `json:"url2"`
	Debug     bool   `json:"debug"`
	OnDemand  bool   `json:"on_demand"`
}

// save stream info
func HTTPAPIStreamSave(c *gin.Context) {
	log.Println("HTTPAPIStreamSave start...")
	authHeader := c.GetHeader("Authorization")
	expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("rino:ese"))

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

	log.Printf("Received: suuid(%s) name(%s) url(%s) debug (%t) ondemand (%t)\n",
		saveParam.Suuid,
		saveParam.Name,
		saveParam.RtspUrl,
		saveParam.Debug,
		saveParam.OnDemand)

	if saveParam.NewStream { //add parameter
		if gStreamListInfo.exist(saveParam.Suuid) {
			log.Println("ApplyStream().. error : already exits")
			c.JSON(http.StatusConflict, gin.H{"status": "failure"})
			return
		}
	} else { //edit/change parameter
		if !gStreamListInfo.exist(saveParam.Suuid) {
			log.Println("ApplyStream().. error : unknown stream id")
			c.JSON(http.StatusNotFound, gin.H{"status": "failure"})
			return
		}
	}

	if gStreamListInfo.ApplyStream(&saveParam) {
		gStreamListInfo.SaveList()
		log.Println("HTTPAPIStreamSave: success")
		c.JSON(http.StatusOK, gin.H{"status": "success"})

	} else {
		log.Println("HTTPAPIStreamSave: failure")
		c.JSON(http.StatusFailedDependency, gin.H{"status": "failure"})

	}
	log.Println("HTTPAPIStreamSave: end")
}

func HTTPAPIServerStreamUploadList(c *gin.Context) {
	log.Println("HTTPAPIServerStreamUploadList start...")
	authHeader := c.GetHeader("Authorization")
	expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("rino:ese"))

	if authHeader != expectedAuth {
		log.Println("HTTPAPIServerStreamUploadList error: Unau thorized...")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var newStreams StreamsMAP
	if err := c.ShouldBindJSON(&newStreams); err != nil {
		log.Println("HTTPAPIServerStreamUploadList error: Invalid JSON format...")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}

	log.Println("HTTPAPIStreamSave: failure")
	c.JSON(http.StatusFailedDependency, gin.H{"status": "failure"})

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
