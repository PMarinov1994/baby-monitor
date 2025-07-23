let audioContext: AudioContext | null = null;
let dataArray: Uint8Array | null = null;

const canvas = document.getElementById('volumeCanvas') as HTMLCanvasElement;
const ctx = canvas.getContext('2d')!;
let isPlaying = false;
const noisePoints: number[] = Array.from({ length: 1000 }, () => 0);

let audioGainScale = 10;

export function setAudioGainValue(gain: number) {
    audioGainScale = gain;
}

export function drawVolume(volume: number): void {
    if (!isPlaying) return;

    noisePoints.shift();
    noisePoints.push(volume);

    ctx.fillStyle = 'black';
    ctx.fillRect(0, 0, canvas.width, canvas.height);

    ctx.fillStyle = 'lime';
    noisePoints.forEach((value, index) => {
        ctx.fillRect(index, canvas.height - value, 2, 2);
    });

    ctx.stroke();
}

export function getAudioContext(): AudioContext {
    if (audioContext === null) {
        audioContext = new (window.AudioContext || (window as any).webkitAudioContext)({ sampleRate: 48000 });
    }

    return audioContext;
}

export function clearAudioContext() {
    if (audioContext) {
        audioContext.close().then(() => {
            console.log('AudioContext closed');
            audioContext = null;
        });

    }
}

export function calculateRMSVolume(data: Float32Array): number {
    // Calculate RMS: sqrt(sum(x[i]^2) / N)
    const gain = audioGainScale;
    let sumSquares = 0;
    for (let i = 0; i < data.length; i++) {
        let amplifiedSample = data[i] * gain;
        sumSquares += amplifiedSample * amplifiedSample;
    }
    let rmsVolume = Math.sqrt(sumSquares / data.length);

    // Scale to 0–400 and convert to integer
    return Math.round(rmsVolume * canvas.height);
}

export function setIsPlaying(value: boolean): void {
    isPlaying = value;
}

export { audioContext, dataArray }
