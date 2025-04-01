
let video_box = null;
let webrtc_svraddr = null;
let webrtc_source_id = null;
let webrtc_api_get_codec = "/stream/codec/"
let webrtc_api_set_remotesdp = "/stream/receiver/"
let webrtc_urlscheme = "http://"
let webrtc_stunaddr = "";
//"stun:" + "127.0.0.1:2222"
//"stun.l.google.com:19302"



let printToPage = msg => {
  let logbox = document.getElementById('webrtcPlayLog');
  if (logbox) {
    let now = new Date();
    let dateString = now.toLocaleString();
    logbox.innerHTML += `[${dateString}] ${msg}<br>`;
  }
  console.log("   '" + msg + "'");
}

async function play_pause_video() {
  if (video_box.paused) {
    printToPage("curr:paused->");
    video_box.play();
    printToPage("now:playing");
  }
};
///////////////////////////////////////////////////////////////////////////////////////////////////

let webrtc_config = {
  // iceServers: [{
  //   urls: [webrtc_stunaddr]
  // }]
};
let webrtc_stream = new MediaStream();
let webrtc_pc = null;
let webrtc_timer10minId = null;

///////////////////////////////////////////////////////////////////////////////////////////////////
function disconnect_webrtc_peer() {
  if (webrtc_stream) {
    webrtc_stream.getTracks().forEach(
      track => {
        printToPage("close..curr track:" + track.label + "  ,   " + track.kind);
        webrtc_stream.removeTrack(track);
        track.stop();
      });
  }
  else {
    webrtc_stream = new MediaStream();
  }

  if (webrtc_pc) {
    webrtc_pc.close();
    webrtc_pc = null;
  }
}

function init_webrtc_peer() {

  disconnect_webrtc_peer();
  {
    /* if (webrtc_pc) {
      if (webrtc_pc.connectionState !== "closed") {
        printToPage("retry: notclosed return");
        return false;
      }
      return true;
    } */
  }

  webrtc_pc = new RTCPeerConnection(webrtc_config);
  {

    webrtc_pc.oniceconnectionstatechange = (event) => {
      //console.log("-->", event);
      printToPage("ice:" + webrtc_pc.iceConnectionState);
    };
    webrtc_pc.onicegatheringstatechange = (event) => {
      //console.log("-->", event);
      printToPage("ice_: " + webrtc_pc.iceGatheringState);
    };
    webrtc_pc.ondatachannel = (event) => {
      //console.log("-->", event);
      printToPage("ondatachannel: " + event);
    };
    webrtc_pc.onpeeridentity = (event) => {
      printToPage("Peer identity: " + event.assertion);
    };
    webrtc_pc.onconnectionstatechange = (event) => {
      //console.log("-->", event);
      printToPage(webrtc_pc.connectionState);
      if (webrtc_pc.connectionState === "disconnected" || webrtc_pc.connectionState === "failed") {
        printToPage("retry");
        restartWebrtcPlayer();
      }
      //else if (webrtc_pc.connectionState === "connected") {
      //play_pause_video();
      //}
    };

  } //end of RTCPeerConnection

  return true;
}


function connect_webrtc_peer() {
  webrtc_pc.onnegotiationneeded = async function (event) {
    console.log("-->", event);
    let offer = await webrtc_pc.createOffer();
    await webrtc_pc.setLocalDescription(offer);
    getRemoteSdp();
  };

  webrtc_pc.onsignalingstatechange = (event) => {
    //console.log("-->", event);
    printToPage("signal: " + webrtc_pc.signalingState);
  };

  webrtc_pc.ontrack = (event) => {
    console.log("-->", event);
    webrtc_stream.getTracks().forEach(track => {
      printToPage("ontrack-> remove old track:" + track.label + "  ,   " + track.kind);
      webrtc_stream.removeTrack(track); //??PYM_TEST_00000 audio track 까지 추가 될 경우 ontrack 이벤트가 2번 발생함으로 새로운 다른 track까지 지워지는 문제가 있음
      track.stop();
    });
    webrtc_stream.addTrack(event.track);
    printToPage(event.streams.length + "byte(s) delivered");
  };

  getCodecInfo();
}

//////////////////////////////////////////////////////////////////////////////////////////////////////
//$(document).ready(function () {
//   console.log('---inner_webrtcPlayClient.js start----');
//   printToPage("ready addclass 0303");
//   openWebrtcPlayer($('#media_svr_address').val(), $('#suuid').val(), videoElem);
//});
function closeWebrtcPlayer() {
  if (webrtc_timer10minId) {
    clearInterval(webrtc_timer10minId);
    webrtc_timer10minId = null;
  }
  disconnect_webrtc_peer();

  video_box.pause();
  video_box.src = "";
  video_box.load();
}

function openWebrtcPlayer(in_webrtc_svraddr, in_suuid, in_videoElem) {
  webrtc_svraddr = in_webrtc_svraddr;
  if (!webrtc_svraddr) {
    printToPage("no media server address");
    return;
  }

  webrtc_source_id = in_suuid;
  if (!webrtc_source_id) {
    printToPage("no uuid");
    return;
  }

  video_box = in_videoElem;
  if (!video_box) {
    printToPage("no videoElem");
    return;
  }
  if (!video_box.srcObject) {
    video_box.srcObject = webrtc_stream;
  }

  if (webrtc_timer10minId == null) {
    const oneminute = 60 * 1000;
    //??PYM_TEST_00000 
    webrtc_timer10minId = setInterval(restartWebrtcPlayer, 1 * oneminute);
    printToPage("____________________timer " + 10 + "minute(s)");
  }

  restartWebrtcPlayer();
}

async function restartWebrtcPlayer() {
  printToPage("----- restartWebrtcPlayer() ----- ");
  printToPage("://" + webrtc_svraddr + " / " + webrtc_source_id);

  init_webrtc_peer();
  connect_webrtc_peer();
}


//////////////////////////////////////////////////////////////////////////////////////////////////////
function getCodecInfo() {  //get /stream/codec/id
  //console.log("getCodecInfo()...");
  printToPage("get /stream/codec/");
  $.get(webrtc_urlscheme + webrtc_svraddr + webrtc_api_get_codec + webrtc_source_id
    , function (data) {
      printToPage("resp received: get /stream/codec/");
      try { data = JSON.parse(data); }
      catch (e) { console.log(e); }
      finally {
        $.each(data
          , function (index, value) {
            webrtc_pc.getTransceivers().forEach((transceiver, index) => { transceiver.stop(); });
            webrtc_pc.addTransceiver(value.Type, { 'direction': 'sendrecv' })
            webrtc_pc.getTransceivers().forEach((transceiver, index) => { printToPage(`Transceiver ${index + 1}:`, transceiver); });//??PYM_TEST_00000
          }
        )
      }
    }
  );
}

function getRemoteSdp() { //post /stream/receiver/id
  //console.log("getRemoteSdp()...");
  printToPage("post /stream/receiver/");
  $.post(webrtc_urlscheme + webrtc_svraddr + webrtc_api_set_remotesdp + webrtc_source_id
    , { suuid: webrtc_source_id, data: btoa(webrtc_pc.localDescription.sdp) }
    , function (data) {
      printToPage("resp received: post /stream/receiver/");
      try { webrtc_pc.setRemoteDescription(new RTCSessionDescription({ type: 'answer', sdp: atob(data) })) }
      catch (e) { console.warn(e); }
    }
  );
}
