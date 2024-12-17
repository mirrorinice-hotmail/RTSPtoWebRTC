
let video_box = null;
let webrtc_svraddr = null;
let webrtc_source_id = null;
let webrtc_api_get_codec = "/stream/codec/"
let webrtc_api_set_remotesdp = "/stream/receiver/"
let webrtc_urlscheme = "http://"
let webrtc_stunaddr = "";//"stun:" + "127.0.0.1:2222"
//"stun.l.google.com:19302"



let printToPage = msg => {
  let now = new Date();
  let dateString = now.toLocaleString();
  document.getElementById('div').innerHTML += `[${dateString}] ${msg}<br>`;
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

let config = {
  // iceServers: [{
  //   urls: [webrtc_stunaddr]
  // }]
};
let stream = new MediaStream();
let pc = null;

///////////////////////////////////////////////////////////////////////////////////////////////////
function disconnect_webrtc_peer() {
  if (pc) {
    stream.getTracks().forEach(track => { 
      {
        printToPage("curr track:" + track.label + "  ,   " + track.kind);
        stream.removeTrack(track);
        track.stop();
      }
    });
    pc.close();
    pc = null;
  }
}

function connect_webrtc_peer() {
  if (pc && pc.connectionState !== "closed") {
    printToPage("retry: notclosed return");
    return;
  }

  if (pc === null) {
    pc = new RTCPeerConnection(config);

    pc.oniceconnectionstatechange = (event) => {
      //console.log("-->", event);
      printToPage("ice:" + pc.iceConnectionState);
    };
    pc.onicegatheringstatechange = (event) => {
      //console.log("-->", event);
      printToPage("ice_: " + pc.iceGatheringState);
    };
    pc.onsignalingstatechange = (event) => {
      //console.log("-->", event);
      printToPage("signal: " + pc.signalingState);
    };
    pc.ondatachannel = (event) => {
      console.log("-->", event);
    };
    pc.onpeeridentity = (event) => {
      printToPage("Peer identity: " + event.assertion);
    };
    pc.onconnectionstatechange = (event) => {
      //console.log("-->", event);
      printToPage(pc.connectionState);
      if (pc.connectionState === "disconnected" || pc.connectionState === "failed") {
        printToPage("retry");
        myGoStream();
      }
      //else if (pc.connectionState === "connected") {
      //play_pause_video();
      //}
    };

    pc.onnegotiationneeded = async function (event) {
      console.log("-->", event);
      let offer = await pc.createOffer();
      await pc.setLocalDescription(offer);
      getRemoteSdp();
    };

    pc.ontrack = (event) => {
      console.log("-->", event);
      stream.getTracks().forEach(track => { 
          printToPage("curr track:" + track.label + "  ,   " + track.kind);
          stream.removeTrack(track);
          track.stop();
      });
      stream.addTrack(event.track);
      if(!video_box.srcObject) video_box.srcObject = stream;
      printToPage(event.streams.length + "byte(s) delivered");
    };
  }

}

//////////////////////////////////////////////////////////////////////////////////////////////////////
$(document).ready(function () {
  console.log('---start----');
  printToPage("ready addclass");
  //$('#suuid').addClass('active');
  //myGoStream();
});

function openStream(in_webrtc_svraddr, in_suuid, in_videoElem) {

  //??PYM_TEST_00000 setInterval(myGoStream, 1000 * 3600);
 // printToPage("____________________1min ");

  if (in_webrtc_svraddr) {
    webrtc_svraddr = in_webrtc_svraddr;
  }
  if (!webrtc_svraddr) {
    printToPage("no media server address");
    return;
  }

  if (in_suuid) {
    webrtc_source_id = in_suuid;
  }
  if (!webrtc_source_id) {
    printToPage("no uuid");
    return;
  }

  if (in_videoElem) {
    video_box = in_videoElem;
  }
  if (!video_box) {
    printToPage("no videoElem");
    return;
  }

  myGoStream(in_webrtc_svraddr, in_suuid, in_videoElem);

}

async function myGoStream(in_webrtc_svraddr, in_suuid, in_videoElem) {
  printToPage("-----------myGoStream()...10.10.1.183-------- ");
  printToPage("server:" + webrtc_svraddr + ", id:" + webrtc_source_id);

  disconnect_webrtc_peer();
  connect_webrtc_peer();
  getCodecInfo();
}


//////////////////////////////////////////////////////////////////////////////////////////////////////
function getCodecInfo() {  //get /stream/codec/id
  console.log("getCodecInfo()...");
  $.get(webrtc_urlscheme + webrtc_svraddr + webrtc_api_get_codec + webrtc_source_id
    , function (data) {
      try { data = JSON.parse(data); }
      catch (e) { console.log(e); }
      finally {
        $.each(data
          , function (index, value) { 
            transceivers = pc.getTransceivers(); //??PYM_TEST_00000
            transceivers.forEach((transceiver, index) => { transceiver.stop(); });
            pc.addTransceiver(value.Type, { 'direction': 'sendrecv' }) 
            pc.getTransceivers().forEach((transceiver, index) => { printToPage(`Transceiver ${index + 1}:`, transceiver); });//??PYM_TEST_00000
          }
        )
      }
    }
  );
}

function getRemoteSdp() { //post /stream/receiver/id
  console.log("getRemoteSdp()...");
  $.post(webrtc_urlscheme + webrtc_svraddr + webrtc_api_set_remotesdp + webrtc_source_id
    , { suuid: webrtc_source_id, data: btoa(pc.localDescription.sdp) }
    , function (data) {
      try { pc.setRemoteDescription(new RTCSessionDescription({ type: 'answer', sdp: atob(data) })) }
      catch (e) { console.warn(e); }
    }
  );
}
