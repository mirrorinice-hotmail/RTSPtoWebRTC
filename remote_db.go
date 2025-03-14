package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

const (
	CCTVLISTMGR_END = iota + 1
	CCTVLISTMGR_UPDATE
)

const (
	test_host      = "localhost"
	test_port      = 5432
	test_user      = "postgres"
	test_pw        = "rino1234"
	test_dbname    = "test_rino_cctv_list"
	test_tablename = "tbl_cctv_info"
)

const (
	col_mgr_no      = "mgr_no"
	col_ip_addr     = "ip_addr"
	col_port_num    = "port_num"
	col_stream_path = "stream_path"
	col_stream_pw   = "stream_pw"
	col_cctv_nm     = "cctv_nm"
	col_addr1       = "addr1"
	col_addr2       = "addr2"
	col_serial_num  = "serial_num"
	col_manager_nm  = "manager_nm"
	col_stream_id   = "stream_id"
	col_rtsp_01     = "rtsp_01"
	col_rtsp_02     = "rtsp_02"
)

type tCctvListMgr struct {
	Name     string
	DbmsInfo DbmsST
	Comm_sig chan int
	Done_sig chan struct{}
}

func (obj *tCctvListMgr) init(dbmsInfo *DbmsST) {
	obj.Name = "CctvListMgr"
	obj.DbmsInfo = *dbmsInfo
	obj.Comm_sig = make(chan int, 10)
	obj.Done_sig = make(chan struct{}, 1)
}

var gCctvListMgr tCctvListMgr

func (obj *tCctvListMgr) db_open() *sql.DB {
	psinfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", obj.DbmsInfo.Host, obj.DbmsInfo.Port, obj.DbmsInfo.User, obj.DbmsInfo.Pass, obj.DbmsInfo.Dbname)

	db, err := sql.Open("postgres", psinfo)
	if err != nil {
		fmt.Println(obj.Name, ": DB connection failure")
	} else {
		fmt.Println(obj.Name, ": DB connection success")
	}

	return db
}

func (obj *tCctvListMgr) update_stream_list() bool {
	remote_db := obj.db_open()
	if remote_db == nil {
		return false
	}
	defer remote_db.Close()

	query := "SELECT " +
		col_stream_id + "," +
		col_rtsp_01 + "," +
		//col_rtsp_02 + "," +
		col_cctv_nm +
		" FROM " + obj.DbmsInfo.TableName
	fmt.Printf("sql : query(%s)\n", query)
	rows, err := remote_db.Query(query)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	newStreams := makeTemporalStreams(rows)
	isListChanged := gStreamListInfo.apply_to_list(newStreams)
	return isListChanged
}

func (obj *tCctvListMgr) db_add(db *sql.DB, in_no string) bool {
	const name = "cctvlist_mgr"
	if in_no == "" {
		return false
	}
	query_action := "INSERT INTO"
	val_mgr_no := in_no
	val_ip_addr := "10.10.0." + val_mgr_no
	val_port_num := "5432"
	val_stream_path := "rtsp://10.10.0.12:5564/" + val_mgr_no + "/stream01"
	val_cctv_nm := "CCTV_" + val_mgr_no
	val_serial_num := "10010" + val_mgr_no
	val_stream_id := "cctv002"
	val_rtsp_01 := "rtsp://210.99.70.120:1935/live/cctv001.stream"
	val_rtsp_02 := "rtsp://"

	query := query_action + " " + obj.DbmsInfo.TableName +
		" (" +
		col_mgr_no + ", " +
		col_ip_addr + ", " +
		col_port_num + ", " +
		col_cctv_nm + ", " +
		col_stream_path + ", " +
		col_serial_num + ", " +
		col_stream_id + ", " +
		col_rtsp_01 + ", " +
		col_rtsp_02 +
		") " +
		" VALUES ($1, $2, $3, $4, $5, $6 , $7, $8, $9) ;"

	log.Println(name, ": db add(", query, ")")
	_, err := db.Exec(query,
		val_mgr_no,
		val_ip_addr,
		val_port_num,
		val_cctv_nm,
		val_stream_path,
		val_serial_num,
		val_stream_id,
		val_rtsp_01,
		val_rtsp_02,
	)

	return obj.db_result_print(err, query_action)
}

func (obj *tCctvListMgr) db_update(db *sql.DB, in_no string) bool {
	const name = "cctvlist_mgr"
	if in_no == "" {
		return false
	}

	query_action := "UPDATE"
	val_mgr_no := in_no
	val_cctv_nm := "CCTV_" + val_mgr_no
	query := query_action + " " + obj.DbmsInfo.TableName +
		" SET " + col_cctv_nm + " = $1" +
		" WHERE " + col_mgr_no + " = $2 ;"
	_, err := db.Exec(query,
		val_cctv_nm,
		val_mgr_no)

	return obj.db_result_print(err, query_action)
}

func (obj *tCctvListMgr) db_delete(db *sql.DB, in_no string) bool {
	const name = "cctvlist_mgr"
	if in_no == "" {
		return false
	}
	query_action := "DELETE FROM"
	val_mgr_no := in_no
	query := query_action + " " + obj.DbmsInfo.TableName +
		" WHERE " + col_mgr_no + " = $1;"
	_, err := db.Exec(query,
		val_mgr_no)

	return obj.db_result_print(err, query_action)
}

func (obj *tCctvListMgr) db_read(db *sql.DB, in_no string) bool {
	const name = "cctvlist_mgr"
	if in_no == "" {
		return false
	}

	query_action := "SELECT"

	query := query_action + " " +
		col_mgr_no + "," + col_stream_path + "," + col_cctv_nm +
		" FROM " + obj.DbmsInfo.TableName
	if in_no != "all" {
		val_mgr_no := in_no
		query = query +
			" WHERE " + col_mgr_no + " = '" + val_mgr_no + "' ;"
	}
	fmt.Printf("%s : query(%s)\n", name, query)
	rows, err := db.Query(query)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		var val_mgr_no string
		var val_stream_path string
		var val_cctv_num string
		err := rows.Scan(&val_mgr_no, &val_stream_path, &val_cctv_num)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s : mgr_no(%s), path(%s), name(%s)\n", name, val_mgr_no, val_stream_path, val_cctv_num)
	}

	return obj.db_result_print(err, query_action)
}

func (obj *tCctvListMgr) db_result_print(err error, in_queryaction string) bool {
	const name = "cctvlist_mgr"
	if err != nil {
		fmt.Println(name, ": err(", err.Error(), ")")
		return false
	} else {
		fmt.Println(name, ": ", in_queryaction, ` success!`)
		return true
	}
}

// ///////////////////////////////////////////////////////////////////////////////

func (obj *tCctvListMgr) request_stop_and_wait() {
	obj.Comm_sig <- CCTVLISTMGR_END
	<-obj.Done_sig
}

func (obj *tCctvListMgr) request_updatelist() {
	obj.Comm_sig <- CCTVLISTMGR_UPDATE
}

func (obj *tCctvListMgr) start() (ot_result int) {
	const name = "cctvlist_mgr"
	log.Println(name, ": Started")
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(name, ": recovered from panic:", r)
		}
		defer func() {
			if r := recover(); r != nil {
				fmt.Println(name, ": recovered from panic: 'done_sig'", r)
			}
			log.Println(name, ": stopped")
		}()

		obj.Done_sig <- struct{}{}
	}()

	cont := true
	for cont {
		switch <-obj.Comm_sig {
		case CCTVLISTMGR_END:
			log.Println(name, ": received 'end'")
			cont = false
		case CCTVLISTMGR_UPDATE:
			log.Println(name, ": received 'update'")
			obj.updateList()
		}
	}

	return 0
}

func (obj *tCctvListMgr) updateList() bool {
	isListChanged := obj.update_stream_list()
	if isListChanged {
		gConfig.SaveConfig()
		go restart()
		return true
	} else {
		return false
	}
}

func makeTemporalStreams(rows *sql.Rows) *StreamsMAP {

	var newStreamsList = make(StreamsMAP)
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("makeTemporalStreams", ": recovered from panic:", r)
		}
	}()

	for rows.Next() {
		var val_stream_id, val_rtsp_01, val_cctv_nm string
		err := rows.Scan(&val_stream_id, &val_rtsp_01, &val_cctv_nm)
		if err != nil {
			panic(err)
		}
		fmt.Printf("stream list: stream_id(%s), rtsp_01(%s), cctv_nm(%s)\n",
			val_stream_id, val_rtsp_01, val_cctv_nm)

		tmpStream := StreamST{
			Uuid:         val_stream_id,
			Name:         val_stream_id,
			Channels:     make(ChannelMAP),
			URL:          val_rtsp_01,
			Status:       false,
			OnDemand:     false,
			DisableAudio: true,
			Debug:        false,
			Codecs:       nil,
			Cl:           make(AvqueueMAP),
			RunLock:      false,
			//msgStop:      make(chan struct{}),
		}
		tmpStream.Channels["0"] = ChannelST{}
		newStreamsList[val_stream_id] = tmpStream
	}
	return &newStreamsList
}
