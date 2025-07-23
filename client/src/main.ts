import { wsStartAudio, wsStopAudio } from "./ws-audio-script";

window.addEventListener('DOMContentLoaded', () => {
    const btnStart = document.getElementById('btnStart');

    if (btnStart instanceof HTMLButtonElement)
        btnStart.onclick = wsStartAudio;

    const btnStop = document.getElementById('btnStop');

    if (btnStop instanceof HTMLButtonElement)
        btnStop.onclick = wsStopAudio;
});
