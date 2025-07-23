import { drawVolume, calculateRMSVolume, setIsPlaying, getAudioContext, clearAudioContext } from './common';

let ws: WebSocket | null = null;

export function wsStartAudio(): void {
    setIsPlaying(true);

    const audioContext = getAudioContext();

    ws = new WebSocket('ws://192.168.100.140:8080/audio');

    ws.onopen = () => {
        console.log('WebSocket connected');
    };

    ws.onmessage = (event: MessageEvent) => {
        const data = event.data as Blob;
        data.arrayBuffer().then((buffer) => {
            const float32Array = new Float32Array(buffer);

            const volume = calculateRMSVolume(float32Array);
            drawVolume(volume);

            const audioBuffer = audioContext.createBuffer(1, float32Array.length, audioContext.sampleRate);
            const channelData = audioBuffer.getChannelData(0);

            for (let i = 0; i < float32Array.length; i++) {
                channelData[i] = float32Array[i];
            }

            const source = audioContext.createBufferSource();
            source.buffer = audioBuffer;
            source.connect(audioContext.destination);
            source.start();
        });
    };

    ws.onerror = (error: Event) => {
        console.error('WebSocket error:', error);
    };

    ws.onclose = () => {
        console.log('WebSocket connection closed');
    };
}

export function wsStopAudio(): void {
    setIsPlaying(false);

    if (ws) {
        console.log("Closing websocket");
        ws.close();
        ws = null;
    }

    clearAudioContext();
}
