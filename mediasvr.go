package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

// WebSocket 업그레이더
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handleConnection(w http.ResponseWriter, r *http.Request) {
	// HTTP 연결을 WebSocket으로 업그레이드
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("WebSocket 업그레이드 실패:", err)
		return
	}
	defer conn.Close()

	fmt.Println("클라이언트 연결됨")

	for {
		// 클라이언트 메시지 읽기
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("메시지 읽기 실패:", err)
			break
		}

		fmt.Printf("받은 메시지: %s\n", msg)

		// 클라이언트에게 응답 보내기
		err = conn.WriteMessage(websocket.TextMessage, []byte("서버 응답: "+string(msg)))
		if err != nil {
			fmt.Println("응답 전송 실패:", err)
			break
		}
	}
}

func main_mediasvr_exam() {
	http.HandleFunc("/ws", handleConnection)
	fmt.Println("WebSocket 서버 실행 중...")
	http.ListenAndServe(":8080", nil)
}
