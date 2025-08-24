const SIZE = 50

let last_media_time: number, last_frame_num: number, fps: number;
let fps_rounder: number[] = new Array(SIZE).fill(0)
let frame_not_seeked = true;

function get_fps_average() {
    return fps_rounder.reduce((a, b) => a + b) / fps_rounder.length;
}

export function monitorFPS(video: HTMLVideoElement, label: HTMLParagraphElement) {
    function ticker(_: DOMHighResTimeStamp, metadata: VideoFrameCallbackMetadata) {
        const now = video.currentTime
        var media_time_diff = Math.abs(now - last_media_time);
        var frame_num_diff = Math.abs(metadata.presentedFrames - last_frame_num);
        var diff = media_time_diff / frame_num_diff;

        if (
            diff &&
            diff < 1 &&
            frame_not_seeked &&
            video.playbackRate === 1
        ) {
            fps_rounder.shift()
            fps_rounder.push(diff);

            fps = Math.round(1 / get_fps_average());
            label.textContent = "FPS: " + fps;
        }
        frame_not_seeked = true;
        last_media_time = now;
        last_frame_num = metadata.presentedFrames;
        video.requestVideoFrameCallback(ticker);
    }

    video.requestVideoFrameCallback(ticker);

    video.addEventListener("seeked", function() {
        fps_rounder.pop();
        frame_not_seeked = false;
    });
}

