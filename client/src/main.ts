import { monitorFPS } from "./streamMonitor";
import { connectToSender } from "./webRTC";
import { wsConnect } from "./webSocket";

const videoElem = document.getElementById('remoteVideo') as HTMLVideoElement;
const videoStat = document.getElementById('videoStats') as HTMLParagraphElement;

const soundSettingsOpenBtn = document.getElementById('soundSettingsOpenBtn') as HTMLButtonElement;
const soundSettingsCloseBtn = document.getElementById('soundSettingsCloseBtn') as HTMLButtonElement;
const soundSettingsDiv = document.getElementById('soundSettings') as HTMLDivElement;

soundSettingsOpenBtn.addEventListener('click', () => {
    soundSettingsDiv.style.height = "100%"
});

soundSettingsCloseBtn.addEventListener('click', () => {
    soundSettingsDiv.style.height = "0%"
})

window.addEventListener('DOMContentLoaded', () => {
    connectToSender(videoElem).then(() => {
        console.log("Successfull connection")
        monitorFPS(videoElem, videoStat)
    });
    wsConnect();
});

videoElem.addEventListener("fullscreenchange", () => {

})
