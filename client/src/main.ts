import { setAudioGainValue } from "./common";
import { wsStartAudio, wsStopAudio } from "./ws-audio-script";

window.addEventListener('DOMContentLoaded', () => {
    const btnStart = document.getElementById('btnStart') as HTMLButtonElement;
    btnStart.onclick = () => wsStartAudio();

    const btnStop = document.getElementById('btnStop') as HTMLButtonElement;
    btnStop.onclick = () => wsStopAudio();

    const slider = document.getElementById('sensitivitySlider') as HTMLInputElement;
    slider.oninput = () => {
        setAudioGainValue(parseInt(slider.value, 10));
    };
});
