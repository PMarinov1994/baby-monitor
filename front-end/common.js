let audioContext = null
// let analyser = null
let dataArray = null
let canvas = document.getElementById('volumeCanvas');
let ctx = canvas.getContext('2d');
let isPlaying = false
let noisePoints = Array.from({ length: 1000 }, () => 0);

let slider_val = 10


// Slider event listener
const slider = document.getElementById('sensitivitySlider');
slider.addEventListener('input', (event) => {
    slider_val = event.target.value
});



// Visualize
function drawVolume(volume) {
    // requestAnimationFrame(drawVolume);

    if (!isPlaying)
        return

    // analyser.getByteFrequencyData(dataArray);
    // const average = dataArray.reduce((a, b) => a + b) / dataArray.length;

    noisePoints.shift()
    noisePoints.push(volume)

    ctx.fillStyle = 'black';
    ctx.fillRect(0, 0, canvas.width, canvas.height);

    ctx.fillStyle = 'lime';

    noisePoints.forEach((value, index) => {
        ctx.fillRect(index, canvas.height - value, 2, 2);
    });

    ctx.stroke();
}



function calculateRMSVolume(float32Array) {
    // Calculate RMS: sqrt(sum(x[i]^2) / N)
    const gain = slider_val;
    let sumSquares = 0;
    for (let i = 0; i < float32Array.length; i++) {
        let amplifiedSample = float32Array[i] * gain;
        sumSquares += amplifiedSample * amplifiedSample;
    }
    let rmsVolume = Math.sqrt(sumSquares / float32Array.length);

    // Scale to 0–400 and convert to integer
    return Math.round(rmsVolume * canvas.height);
}
