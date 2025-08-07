import { monitorFPS } from "./streamMonitor";
import { connectToSender } from "./webRTC";
import { wsConnect } from "./webSocket";

window.addEventListener('DOMContentLoaded', () => {
    const videoElem = document.getElementById('remoteVideo') as HTMLVideoElement
    const videoStat = document.getElementById('videoStats') as HTMLParagraphElement

    connectToSender(videoElem).then(() => {
        console.log("Successfull connection")
        monitorFPS(videoElem, videoStat)
    })
    wsConnect()
});
