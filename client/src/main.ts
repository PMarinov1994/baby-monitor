import { wsConnect } from "./webSocket";
import { Watchdog } from './watchdog.js';
import { connectToSender } from './webRTC.js';

const CHECK_VIDEO_MS = 1000
const ALARM_VOLUME = 1 // from 0.0 to 1.0

let intervalId: number | undefined = undefined
let frameProc: boolean = false

const audio = new Audio('/alarm.mp3');
audio.volume = ALARM_VOLUME

const watchdog = new Watchdog(CHECK_VIDEO_MS * 2, () => {
    audio.play()
}, () => {
    audio.pause()
})

const videoElem = document.getElementById('remoteVideo') as HTMLVideoElement;
videoElem.addEventListener('play', () => {
    console.log("Video Play")

    intervalId = setInterval(() => {
        if (frameProc)
            watchdog.reset()
        frameProc = false
    }, CHECK_VIDEO_MS)

    watchdog.start()
    videoElem.requestVideoFrameCallback(processFrame)
});

videoElem.addEventListener('pause', () => {
    console.log("Video Pause")

    clearInterval(intervalId)
    watchdog.stop()
})

const soundSettingsOpenBtn = document.getElementById('soundSettingsOpenBtn') as HTMLButtonElement;
const soundSettingsDiv = document.getElementById('soundSettings') as HTMLDivElement;

soundSettingsOpenBtn.addEventListener('click', () => {
    if (soundSettingsDiv.style.height === "100%")
        soundSettingsDiv.style.height = "0%"
    else
        soundSettingsDiv.style.height = "100%"
});

function processFrame() {
    frameProc = true

    // Continue loop if video is playing
    videoElem.requestVideoFrameCallback(processFrame)
}

window.addEventListener('DOMContentLoaded', () => {
    connectToSender(videoElem).then(() => {
        console.log("WebRTC Connected!")
        wsConnect();
    })
})
