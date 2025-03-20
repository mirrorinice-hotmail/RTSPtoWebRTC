package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

var gCctvListMgr tCctvListMgr

////////////////////////////////////

const (
	CCTVLISTMGR_END = iota + 1
	CCTVLISTMGR_UPDATE
)

// const (
// 	test_host      = "localhost"
// 	test_port      = 5432
// 	test_user      = "postgres"
// 	test_pw        = "rino1234"
// 	test_dbname    = "test_rino_cctv_list"
// 	test_tablename = "tbl_cctv_info"
// )

const (
	col_mgr_no      = "mgr_no"
	col_cctv_ip     = "ip_addr"
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

func (obj *tCctvListMgr) update_stream_list() StreamsMAP {

	var newStream StreamsMAP
	remote_db := obj.db_open()
	if remote_db == nil {
		return newStream
	}
	defer remote_db.Close()

	query := "SELECT " +
		col_stream_id + "," +
		col_rtsp_01 + "," +
		//col_rtsp_02 + "," +
		col_cctv_nm + "," +
		col_cctv_ip +
		" FROM " + obj.DbmsInfo.TableName
	fmt.Printf("sql : query(%s)\n", query)
	rows, err := remote_db.Query(query)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	newStream = makeTemporalStreams(rows)
	return newStream
}

func (obj *tCctvListMgr) db_add_samples() bool {
	remote_db := obj.db_open()
	if remote_db == nil {
		return false
	}
	defer remote_db.Close()

	query_base := "INSERT INTO tbl_cctv_info (mgr_no, ip_addr, port_num, cctv_nm, stream_id, rtsp_02, rtsp_01) "

	for i, val := range indsert_val_list {
		query := query_base + val
		fmt.Printf("%d : query(%s)\n", i, query)
		_, err := remote_db.Query(query)
		if err != nil {
			panic(err)
			return false
		}
	}
	return true
}

/*
func (obj *tCctvListMgr) db_add(db *sql.DB, in_no string) bool {
	const name = "cctvlist_mgr"
	if in_no == "" {
		return false
	}
	query_action := "INSERT INTO"
	val_mgr_no := in_no
	val_cctv_ip := "10.10.0." + val_mgr_no
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
		col_cctv_ip + ", " +
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
		val_cctv_ip,
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
} */

// ///////////////////////////////////////////////////////////////////////////////

func (obj *tCctvListMgr) request_stop_and_wait() {
	obj.Comm_sig <- CCTVLISTMGR_END
	<-obj.Done_sig
}

// func (obj *tCctvListMgr) request_updatelist() {
// 	obj.Comm_sig <- CCTVLISTMGR_UPDATE
// }

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
	newStreams := obj.update_stream_list()
	isListChanged := gStreamListInfo.apply_to_list(newStreams)
	if isListChanged {
		return true
	} else {
		return false
	}
}

func makeTemporalStreams(rows *sql.Rows) StreamsMAP {

	var newStreamsList = make(StreamsMAP)
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("makeTemporalStreams", ": recovered from panic:", r)
		}
	}()

	for rows.Next() {
		var val_stream_id, val_rtsp_01, val_cctv_nm, val_cctv_ip string
		err := rows.Scan(&val_stream_id, &val_rtsp_01, &val_cctv_nm, &val_cctv_ip)
		if err != nil {
			panic(err)
		}
		fmt.Printf("stream list: stream_id(%s), rtsp_01(%s), cctv_nm(%s)\n",
			val_stream_id, val_rtsp_01, val_cctv_nm)

		tmpStream := StreamST{
			Uuid:         val_stream_id,
			CctvName:     val_cctv_nm,
			CctvIp:       val_cctv_ip,
			Channels:     make(ChannelMAP),
			RtspUrl:      val_rtsp_01,
			Status:       false,
			OnDemand:     false,
			DisableAudio: true,
			Debug:        false,
			Codecs:       nil,
			avQue:        make(AvqueueMAP),
			RunLock:      false,
		}
		tmpStream.Channels["0"] = ChannelST{}
		newStreamsList[val_stream_id] = tmpStream
	}
	return newStreamsList
}

var indsert_val_list []string = []string{
	"VALUES ('cctv_1_1_1' , '10.1.1.51' , '5432' , '합천읍 중흥길 25' , 'cctv_1_1_1' , 'rtsp://10.1.1.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.1.1.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_1_2_1' , '10.1.2.51' , '5432' , '합천읍 교동1길 24' , 'cctv_1_2_1' , 'rtsp://10.1.2.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.1.2.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_1_4_1' , '10.1.4.51' , '5432' , '합천읍 신소양1길 4' , 'cctv_1_4_1' , 'rtsp://10.1.4.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.1.4.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_1_6_1' , '10.1.6.51' , '5432' , '합천읍 영창1길 9' , 'cctv_1_6_1' , 'rtsp://10.1.6.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.1.6.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_1_8_1' , '10.1.8.51' , '5432' , '합천읍 서산길 56' , 'cctv_1_8_1' , 'rtsp://10.1.8.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.1.8.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_1_9_1' , '10.1.9.51' , '5432' , '합천읍 충효로 152, 102동 104호' , 'cctv_1_9_1' , 'rtsp://10.1.9.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.1.9.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_1_10_1' , '10.1.10.51' , '5432' , '합천읍 중앙로4길 16' , 'cctv_1_10_1' , 'rtsp://10.1.10.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.1.10.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_2_1_1' , '10.2.1.51' , '5432' , ' 봉산면 서부로 4071' , 'cctv_2_1_1' , 'rtsp://10.2.1.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.2.1.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_2_2_1' , '10.2.2.51' , '5432' , ' 봉산면 영서로 1528' , 'cctv_2_2_1' , 'rtsp://10.2.2.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.2.2.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_3_1_1' , '10.3.1.51' , '5432' , '묘산면 묘산로 163' , 'cctv_3_1_1' , 'rtsp://10.3.1.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.3.1.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_3_2_1' , '10.3.2.51' , '5432' , '묘산면 도옥길 40' , 'cctv_3_2_1' , 'rtsp://10.3.2.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.3.2.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_4_2_1' , '10.4.2.51' , '5432' , '가야면 가야시장로 50-29' , 'cctv_4_2_1' , 'rtsp://10.4.2.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.4.2.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_4_3_1' , '10.4.3.51' , '5432' , '가야면 가야시장로 95' , 'cctv_4_3_1' , 'rtsp://10.4.3.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.4.3.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_5_1_1' , '10.5.1.51' , '5432' , '야로면 가야산로 242-3' , 'cctv_5_1_1' , 'rtsp://10.5.1.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.5.1.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_5_2_1' , '10.5.2.51' , '5432' , '야로면 월광1길 1' , 'cctv_5_2_1' , 'rtsp://10.5.2.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.5.2.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_5_3_1' , '10.5.3.51' , '5432' , '야로면 매촌2길 5' , 'cctv_5_3_1' , 'rtsp://10.5.3.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.5.3.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_6_1_1' , '10.6.1.51' , '5432' , '율곡면 황강옥전로 626' , 'cctv_6_1_1' , 'rtsp://10.6.1.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.6.1.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_6_3_1' , '10.6.3.51' , '5432' , '율곡면 노양길 186-7' , 'cctv_6_3_1' , 'rtsp://10.6.3.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.6.3.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_7_1_1' , '10.7.1.51' , '5432' , '초계면 원당길 42' , 'cctv_7_1_1' , 'rtsp://10.7.1.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.7.1.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_7_3_1' , '10.7.3.51' , '5432' , '초계면 아막재로 37' , 'cctv_7_3_1' , 'rtsp://10.7.3.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.7.3.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_7_4_1' , '10.7.4.51' , '5432' , '초계면 내동3길 11' , 'cctv_7_4_1' , 'rtsp://10.7.4.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.7.4.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_7_5_1' , '10.7.5.51' , '5432' , '초계면 초계중앙로 26-1' , 'cctv_7_5_1' , 'rtsp://10.7.5.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.7.5.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_8_1_1' , '10.8.1.51' , '5432' , '쌍책면 오광대로 129-1' , 'cctv_8_1_1' , 'rtsp://10.8.1.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.8.1.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_8_2_1' , '10.8.2.51' , '5432' , '쌍책면 황강옥전로 1596' , 'cctv_8_2_1' , 'rtsp://10.8.2.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.8.2.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_9_1_1' , '10.9.1.51' , '5432' , '덕곡면 율원1길 19-4' , 'cctv_9_1_1' , 'rtsp://10.9.1.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.9.1.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_9_2_1' , '10.9.2.51' , '5432' , '덕곡면 포두1길 77' , 'cctv_9_2_1' , 'rtsp://10.9.2.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.9.2.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_10_1_1' , '10.10.1.51' , '5432' , '청덕면 가현길 69' , 'cctv_10_1_1' , 'rtsp://10.10.1.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.10.1.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_10_2_1' , '10.10.2.51' , '5432' , '청덕면 초곡길 151-1' , 'cctv_10_2_1' , 'rtsp://10.10.2.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.10.2.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_11_1_1' , '10.11.1.51' , '5432' , '적중면 중부2길 30-6' , 'cctv_11_1_1' , 'rtsp://10.11.1.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.11.1.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_11_2_1' , '10.11.2.51' , '5432' , '적중면 상부1길 30' , 'cctv_11_2_1' , 'rtsp://10.11.2.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.11.2.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_12_1_1' , '10.12.1.51' , '5432' , '대양면 정양2길 45' , 'cctv_12_1_1' , 'rtsp://10.12.1.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.12.1.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_12_2_1' , '10.12.2.51' , '5432' , '대양면 백암1길 17' , 'cctv_12_2_1' , 'rtsp://10.12.2.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.12.2.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_13_1_1' , '10.13.1.51' , '5432' , '쌍백면 외초1길 32' , 'cctv_13_1_1' , 'rtsp://10.13.1.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.13.1.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_13_2_1' , '10.13.2.51' , '5432' , '쌍백면 평구묵골길 377' , 'cctv_13_2_1' , 'rtsp://10.13.2.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.13.2.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_14_1_1' , '10.14.1.51' , '5432' , '삼가면 소오2길 2' , 'cctv_14_1_1' , 'rtsp://10.14.1.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.14.1.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_15_1_1' , '10.15.1.51' , '5432' , '가회면 황매산로 82' , 'cctv_15_1_1' , 'rtsp://10.15.1.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.15.1.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_16_1_1' , '10.16.1.51' , '5432' , '대병면 신성동길 22' , 'cctv_16_1_1' , 'rtsp://10.16.1.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.16.1.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_16_2_1' , '10.16.2.51' , '5432' , '대병면 금객1길 60' , 'cctv_16_2_1' , 'rtsp://10.16.2.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.16.2.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_17_1_1' , '10.17.1.51' , '5432' , '용주면 월평길 161' , 'cctv_17_1_1' , 'rtsp://10.17.1.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.17.1.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_17_2_1' , '10.17.2.51' , '5432' , '용주면 봉기길 26' , 'cctv_17_2_1' , 'rtsp://10.17.2.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.17.2.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_17_3_1' , '10.17.3.51' , '5432' , '용주면 고품1길 12-1' , 'cctv_17_3_1' , 'rtsp://10.17.3.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.17.3.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_17_4_1' , '10.17.4.51' , '5432' , '용주면 가호길 91' , 'cctv_17_4_1' , 'rtsp://10.17.4.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.17.4.51:554/Profile02/media.smp' )",
	"VALUES ('cctv_17_5_1' , '10.17.5.51' , '5432' , '용주면 고품3길 27' , 'cctv_17_5_1' , 'rtsp://10.17.5.50:558/LiveChannel/00/media.smp' , 'rtsp://hcmanager:hap_1000!@10.17.5.51:554/Profile02/media.smp' )",
}
