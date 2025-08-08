import { monitorFPS } from "./streamMonitor";
import { connectToSender } from "./webRTC";
import { wsConnect } from "./webSocket";

const videoElem = document.getElementById('remoteVideo') as HTMLVideoElement;
const videoStat = document.getElementById('videoStats') as HTMLParagraphElement;

const soundSettingsBtn = document.getElementById('soundToggleBtn') as HTMLButtonElement;
const soundSettingsDiv = document.getElementById('soundSettings') as HTMLDivElement;


soundSettingsBtn.addEventListener('click', () => {
    if (soundSettingsDiv.style.display === 'none')
        soundSettingsDiv.style.display = 'block';
    else
        soundSettingsDiv.style.display = 'none';
});

window.addEventListener('DOMContentLoaded', () => {
    connectToSender(videoElem).then(() => {
        console.log("Successfull connection")
        monitorFPS(videoElem, videoStat)
    });
    wsConnect();
});
