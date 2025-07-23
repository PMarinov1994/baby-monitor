let audioContext: AudioContext | null = null;
let dataArray: Uint8Array | null = null;

const canvas = document.getElementById('volumeCanvas') as HTMLCanvasElement;
const ctx = canvas.getContext('2d')!;
let isPlaying = false;
const noisePoints: number[] = Array.from({ length: 1000 }, () => 0);

let slider_val = 10;

const slider = document.getElementById('sensitivitySlider') as HTMLInputElement;
slider.addEventListener('input', (event: Event) => {
    const target = event.target as HTMLInputElement;
    slider_val = parseInt(target.value, 10);
    console.log(slider_val)
});

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

export function calculateRMSVolume(data: Float32Array): number {
    const sumSquares = data.reduce((sum, value) => sum + value * value, 0);
    return Math.sqrt(sumSquares / data.length) * 100;
}

export function setIsPlaying(value: boolean): void {
    isPlaying = value;
}

export { audioContext, dataArray }
