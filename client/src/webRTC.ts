export async function connectToSender(videoEl: HTMLVideoElement) {
    console.log('CONNECTING TO SENDER');

    const pc = new RTCPeerConnection({})

    pc.ontrack = event => {
        // console.log(event.streams)
        // videoEl.srcObject = event.streams[0];
        console.log('Received track:', event);
        if (videoEl.srcObject !== event.streams[0]) {
            videoEl.srcObject = event.streams[0]
        }
    }

    const audioTRansceiver = pc.addTransceiver('audio', { direction: 'recvonly' })
    audioTRansceiver.receiver.jitterBufferTarget = 1

    const videoTransceiver = pc.addTransceiver('video', { direction: 'recvonly' })
    videoTransceiver.receiver.jitterBufferTarget = 1

    const videoCapabilities = RTCRtpReceiver.getCapabilities("video")
    if (videoCapabilities === null) {
        alert("pc.localDescription is null")
        return
    }

    const codecs = videoCapabilities.codecs;
    const preferred = codecs.filter(c => c.mimeType === "video/H264")

    videoTransceiver.setCodecPreferences(preferred);

    try {
        const offer = await pc.createOffer()
        await pc.setLocalDescription(offer);

        if (pc.localDescription === null) {
            alert("pc.localDescription is null")
            return
        }

        console.log("Sending request");
        const response = await fetch('/webRTCFeed', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                sdp: pc.localDescription.sdp,
                type: 'offer',
            })
        });

        if (!response.ok) {
            throw new Error('Failed to send offer');
        }

        console.log("Waiting response...");
        const answer = await response.json();

        console.log("setRemoteDescription");
        // await pc.setRemoteDescription(new RTCSessionDescription(answer))
        await pc.setRemoteDescription(new RTCSessionDescription(
            {
                type: 'answer',
                sdp: answer.sdp
            }));

        for (const candidate of answer.candidates)
            await pc.addIceCandidate(new RTCIceCandidate(candidate));

    } catch (e) {
        alert(e)
    }

}
