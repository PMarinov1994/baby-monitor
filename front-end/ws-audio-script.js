let ws = null

function wsStartAudio() {
    isPlaying = true

    // analyser = audioContext.createAnalyser();
    // analyser.fftSize = 256;

    // dataArray = new Uint8Array(analyser.frequencyBinCount);
    // drawVolume();

    audioContext = new (window.AudioContext || window.webkitAudioContext)({ sampleRate: 48000 });
    ws = new WebSocket('ws://192.168.100.140:8080/audio'); // Replace with your WebSocket URL

    ws.onopen = () => {
        console.log('WebSocket connected');
    };

    ws.onmessage = (event) => {
        event.data.arrayBuffer().then((buffer) => {
            const float32Array = new Float32Array(buffer);

            const volume = calculateRMSVolume(float32Array)
            drawVolume(volume)

            const audioBuffer = audioContext.createBuffer(1, float32Array.length, audioContext.sampleRate);
            const channel = audioBuffer.getChannelData(0);
            for (let i = 0; i < float32Array.length; i++)
                channel[i] = float32Array[i];

            const source = audioContext.createBufferSource();
            source.buffer = audioBuffer;
            source.connect(audioContext.destination);

            // source.connect(analyser);
            // analyser.connect(audioContext.destination);
            source.start();
        });
    };

    ws.onclose = () => {
        console.log('WebSocket disconnected');
    };

    ws.onerror = (error) => {
        console.error('WebSocket error:', error);
    };
}

function wsStopAudio() {
    isPlaying = false
    if (ws) {
        console.log("Closing websocket")
        ws.close();
        ws = null;
    }
    if (audioContext) {
        audioContext.close().then(() => {
            audioContext = null;
            queue = [];
            nextTime = 0;
            console.log('AudioContext closed');
        });
    }
}
