package main

//stream.go : rtsp streaming receiver

import (
	"errors"
	"log"
	"net"
	"time"

	"deepch_vdk/format/rtspv2" //"github.com/deepch/vdk/format/rtspv2"
)

var (
	ErrorStreamExit_NoVideoOnStream = errors.New("stream Exit No Video On Stream")
	ErrorStreamExit_RtspDisconnect  = errors.New("stream Exit Rtsp Disconnect")
	ErrorStreamExit_NoViewer        = errors.New("stream Exit On Demand No Viewer")
	ErrorStreamExit_StopMsgReceived = errors.New("stream Exit Stop message received")
)

func RTSPWorker(msgStop <-chan struct{}, suuid, url string, OnDemand, DisableAudio, Debug bool) error {
	/*	RTSPClient, err := rtspv2.Dial(
		rtspv2.RTSPClientOptions{
			URL: url,
			DisableAudio: DisableAudio,
			DialTimeout: 3 * time.Second,
			ReadWriteTimeout: 3 * time.Second,
			Debug: Debug})*/
	RTSPClient, err := rtspv2.Dial2(
		rtspv2.RTSPClientOptions{
			URL:              url,
			DisableAudio:     DisableAudio,
			DialTimeout:      3 * time.Second,
			ReadWriteTimeout: 3 * time.Second,
			Debug:            Debug},
		"0.0.0.0:0")
	if err != nil {
		return err
	}
	defer RTSPClient.Close()

	if RTSPClient.CodecData != nil {
		gStreamListInfo.setCodec(suuid, RTSPClient.CodecData)
	}
	var AudioOnly bool
	if len(RTSPClient.CodecData) == 1 && RTSPClient.CodecData[0].Type().IsAudio() {
		AudioOnly = true
	}

	//add next TimeOut
	keyTest := time.NewTimer(20 * time.Second)
	clientTest := time.NewTimer(20 * time.Second)
	for {
		select {
		case <-msgStop:
			log.Println("RTSPWorker : ErrorStreamExit_StopMsgReceived")
			return ErrorStreamExit_StopMsgReceived
		case <-clientTest.C:
			if OnDemand {
				if !gStreamListInfo.HasViewer(suuid) {
					return ErrorStreamExit_NoViewer
				} else {
					clientTest.Reset(20 * time.Second)
				}
			}
		case <-keyTest.C:
			return ErrorStreamExit_NoVideoOnStream
		case signals := <-RTSPClient.Signals:
			switch signals {
			case rtspv2.SignalCodecUpdate:
				gStreamListInfo.setCodec(suuid, RTSPClient.CodecData)
			case rtspv2.SignalStreamRTPStop:
				return ErrorStreamExit_RtspDisconnect
			}
		case packetAV := <-RTSPClient.OutgoingPacketQueue:
			if AudioOnly || packetAV.IsKeyFrame {
				keyTest.Reset(20 * time.Second)
			}
			gStreamListInfo.cast(suuid, *packetAV)
		}
	}
}

// ////////////////////////////////////////////////
func DialTimeout_localIp(localIpPort, network, address string, timeout time.Duration) (net.Conn, error) {
	localAddress, err := net.ResolveTCPAddr("tcp", localIpPort)
	if err != nil {
		return nil, err
	}
	d := net.Dialer{Timeout: timeout, LocalAddr: localAddress}
	return d.Dial(network, address)
}

/*
func Dial_localIp(options RTSPClientOptions, localIpPort string) (*RTSPClient, error) {
	client := &RTSPClient{
		headers:             make(map[string]string),
		Signals:             make(chan int, 100),
		OutgoingProxyQueue:  make(chan *[]byte, 3000),
		OutgoingPacketQueue: make(chan *av.Packet, 3000),
		BufferRtpPacket:     bytes.NewBuffer([]byte{}),
		videoID:             -1,
		audioID:             -2,
		videoIDX:            -1,
		audioIDX:            -2,
		options:             options,
		AudioTimeScale:      8000,
	}
	client.headers["User-Agent"] = "Lavf58.76.100"
	err := client.parseURL(html.UnescapeString(client.options.URL))
	if err != nil {
		return nil, err
	}
	conn, err := DialTimeout2(localIpPort, "tcp", client.pURL.Host, client.options.DialTimeout)
	//net.DialTimeout("tcp", client.pURL.Host, client.options.DialTimeout)
	if err != nil {
		return nil, err
	}
	err = conn.SetDeadline(time.Now().Add(client.options.ReadWriteTimeout))
	if err != nil {
		return nil, err
	}
	if client.pURL.Scheme == "rtsps" {
		tlsConn := tls.Client(conn, &tls.gConfig{InsecureSkipVerify: options.InsecureSkipVerify, ServerName: client.pURL.Hostname()})
		err = tlsConn.Handshake()
		if err != nil {
			return nil, err
		}
		conn = tlsConn
	}
	client.conn = conn
	client.connRW = bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	err = client.request(OPTIONS, nil, client.pURL.String(), false, false)
	if err != nil {
		return nil, err
	}
	err = client.request(DESCRIBE, map[string]string{"Accept": "application/sdp"}, client.pURL.String(), false, false)
	if err != nil {
		return nil, err
	}
	for _, i2 := range client.mediaSDP {
		if (i2.AVType != VIDEO && i2.AVType != AUDIO) || (client.options.DisableAudio && i2.AVType == AUDIO) {
			//TODO check it
			if strings.Contains(string(client.SDPRaw), "LaunchDigital") {
				client.chTMP += 2
			}
			continue
		}
		err = client.request(SETUP, map[string]string{"Transport": "RTP/AVP/TCP;unicast;interleaved=" + strconv.Itoa(client.chTMP) + "-" + strconv.Itoa(client.chTMP+1)}, client.ControlTrack(i2.Control), false, false)
		if err != nil {
			return nil, err
		}
		if i2.AVType == VIDEO {
			if i2.Type == av.H264 {
				if len(i2.SpropParameterSets) > 1 {
					if codecData, err := h264parser.NewCodecDataFromSPSAndPPS(i2.SpropParameterSets[0], i2.SpropParameterSets[1]); err == nil {
						client.sps = i2.SpropParameterSets[0]
						client.pps = i2.SpropParameterSets[1]
						client.CodecData = append(client.CodecData, codecData)
					}
				} else {
					client.CodecData = append(client.CodecData, h264parser.CodecData{})
					client.WaitCodec = true
				}
				client.FPS = i2.FPS
				client.videoCodec = av.H264
			} else if i2.Type == av.H265 {
				if len(i2.SpropVPS) > 1 && len(i2.SpropSPS) > 1 && len(i2.SpropPPS) > 1 {
					if codecData, err := h265parser.NewCodecDataFromVPSAndSPSAndPPS(i2.SpropVPS, i2.SpropSPS, i2.SpropPPS); err == nil {
						client.vps = i2.SpropVPS
						client.sps = i2.SpropSPS
						client.pps = i2.SpropPPS
						client.CodecData = append(client.CodecData, codecData)
					}
				} else {
					client.CodecData = append(client.CodecData, h265parser.CodecData{})
				}
				client.videoCodec = av.H265

			} else {
				client.Println("SDP Video Codec Type Not Supported", i2.Type)
			}
			client.videoIDX = int8(len(client.CodecData) - 1)
			client.videoID = client.chTMP
		}
		if i2.AVType == AUDIO {
			client.audioID = client.chTMP
			var CodecData av.AudioCodecData
			switch i2.Type {
			case av.AAC:
				CodecData, err = aacparser.NewCodecDataFromMPEG4AudioConfigBytes(i2.gConfig)
				if err == nil {
					client.Println("Audio AAC bad config")
				}
			case av.OPUS:
				var cl av.ChannelLayout
				switch i2.ChannelCount {
				case 1:
					cl = av.CH_MONO
				case 2:
					cl = av.CH_STEREO
				default:
					cl = av.CH_MONO
				}
				CodecData = codec.NewOpusCodecData(i2.TimeScale, cl)
			case av.PCM_MULAW:
				CodecData = codec.NewPCMMulawCodecData()
			case av.PCM_ALAW:
				CodecData = codec.NewPCMAlawCodecData()
			case av.PCM:
				CodecData = codec.NewPCMCodecData()
			default:
				client.Println("Audio Codec", i2.Type, "not supported")
			}
			if CodecData != nil {
				client.CodecData = append(client.CodecData, CodecData)
				client.audioIDX = int8(len(client.CodecData) - 1)
				client.audioCodec = CodecData.Type()
				if i2.TimeScale != 0 {
					client.AudioTimeScale = int64(i2.TimeScale)
				}
			}
		}
		client.chTMP += 2
	}
	//test := map[string]string{"Scale": "1.000000", "Speed": "1.000000", "Range": "clock=20210929T210000Z-20210929T211000Z"}
	err = client.request(PLAY, nil, client.control, false, false)
	if err != nil {
		return nil, err
	}
	go client.startStream()
	return client, nil
} //end of 'func_Dial_localIp'
*/
