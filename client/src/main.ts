import { connectToSender } from "./webRTC";
import { wsConnect } from "./webSocket";

const videoElem = document.getElementById('remoteVideo') as HTMLVideoElement;

const soundSettingsOpenBtn = document.getElementById('soundSettingsOpenBtn') as HTMLButtonElement;
const soundSettingsDiv = document.getElementById('soundSettings') as HTMLDivElement;

soundSettingsOpenBtn.addEventListener('click', () => {
    if (soundSettingsDiv.style.height === "100%")
        soundSettingsDiv.style.height = "0%"
    else
        soundSettingsDiv.style.height = "100%"
});

window.addEventListener('DOMContentLoaded', () => {
    const pc = new RTCPeerConnection({})
    const dc = pc.createDataChannel("exchange_id")

    connectToSender(pc, videoElem).then(() => {
        console.log("Successfull connection to webRTC.")
        console.log("Connecting to WebSocket.")
        wsConnect(pc, dc);
    });
});
