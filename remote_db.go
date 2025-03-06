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
	host      = "localhost"
	port      = 5432
	user      = "postgres"
	pw        = "rino1234"
	dbname    = "test_rino_cctv_list"
	tablename = "tbl_cctv_info"
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

func db_open() *sql.DB {
	const name = "cctvlist_mgr"
	psinfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, pw, dbname)

	db, err := sql.Open("postgres", psinfo)
	if err != nil {
		fmt.Println(name, ": DB connection failure")
	} else {
		fmt.Println(name, ": DB connection success")
	}

	return db
}

func update_stream_list() {
	remote_db := db_open()
	if remote_db == nil {
		return
	}
	defer remote_db.Close()

	query := "SELECT " +
		col_stream_id + "," +
		col_rtsp_01 + "," +
		//col_rtsp_02 + "," +
		col_cctv_nm +
		" FROM " + tablename
	fmt.Printf("sql : query(%s)\n", query)
	rows, err := remote_db.Query(query)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	gStreamListInfo.update_list(rows)
	/*
		//gStreamListInfo.
		for rows.Next() {
			gStreamListInfo.mutex.Lock()
			defer gStreamListInfo.mutex.Unlock()
			var val_stream_id, val_rtsp_01, val_rtsp_02, val_cctv_nm string
			err := rows.Scan(&val_stream_id, &val_rtsp_01, &val_rtsp_02, &val_cctv_nm)
			if err != nil {
				panic(err)
			}
			fmt.Printf("stream list: stream_id(%s), rtsp_01(%s), rtsp_02(%s) , cctv_nm(%s)\n",
				val_stream_id, val_rtsp_01, val_rtsp_02, val_cctv_nm)

			if tmpStream, ok := (*gStreamListInfo.Streams)[val_stream_id]; ok {
				tmpStream.URL = val_rtsp_01
				(*gStreamListInfo.Streams)[val_stream_id] = tmpStream
			} else {
				tmpStream := StreamST{
					Uuid:         val_stream_id,
					URL:          val_rtsp_01,
					Status:       false,
					OnDemand:     false,
					DisableAudio: true,
					Debug:        false,
					RunLock:      false,
					Codecs:       nil,
					Cl:           make(map[string]avQueue)}
				(*gStreamListInfo.Streams)[val_stream_id] = tmpStream
			}
		}
	*/

}

func db_add(db *sql.DB, in_no string) bool {
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

	query := query_action + " " + tablename +
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

	return db_result_print(err, query_action)
}

func db_update(db *sql.DB, in_no string) bool {
	const name = "cctvlist_mgr"
	if in_no == "" {
		return false
	}

	query_action := "UPDATE"
	val_mgr_no := in_no
	val_cctv_nm := "CCTV_" + val_mgr_no
	query := query_action + " " + tablename +
		" SET " + col_cctv_nm + " = $1" +
		" WHERE " + col_mgr_no + " = $2 ;"
	_, err := db.Exec(query,
		val_cctv_nm,
		val_mgr_no)

	return db_result_print(err, query_action)
}

func db_delete(db *sql.DB, in_no string) bool {
	const name = "cctvlist_mgr"
	if in_no == "" {
		return false
	}
	query_action := "DELETE FROM"
	val_mgr_no := in_no
	query := query_action + " " + tablename +
		" WHERE " + col_mgr_no + " = $1;"
	_, err := db.Exec(query,
		val_mgr_no)

	return db_result_print(err, query_action)
}

func db_read(db *sql.DB, in_no string) bool {
	const name = "cctvlist_mgr"
	if in_no == "" {
		return false
	}

	query_action := "SELECT"

	query := query_action + " " +
		col_mgr_no + "," + col_stream_path + "," + col_cctv_nm +
		" FROM " + tablename
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

	return db_result_print(err, query_action)
}

func db_result_print(err error, in_queryaction string) bool {
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
var cctvlist_mgr_comm_sig = make(chan int, 1)

func cctvlist_mgr_stop() {
	cctvlist_mgr_comm_sig <- CCTVLISTMGR_END
}

func cctvlist_mgr_updatelist() {
	cctvlist_mgr_comm_sig <- CCTVLISTMGR_UPDATE
}

func cctvlist_mgr(done_sig chan<- struct{}) (ot_result int) {
	const name = "cctvlist_mgr"
	log.Println(name, ": cctvlist_mgr start")
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(name, ": recovered from panic:", r)
		}
		log.Println(name, ": stopped")
	}()

	defer func() {
		if r := recover(); r != nil {
			fmt.Println(name, ": recovered from panic:", r)
		}
		done_sig <- struct{}{} //??PYM_TEST_00000
	}()

	cont := true
	for cont {
		switch <-cctvlist_mgr_comm_sig {
		case CCTVLISTMGR_END:
			cont = false
			log.Println(name, ": received 'end'")
		case CCTVLISTMGR_UPDATE:
			log.Println(name, ": received 'update'")
			update_stream_list()
		}
	}

	done_sig <- struct{}{}
	return 0
}
