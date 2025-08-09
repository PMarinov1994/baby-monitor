export async function connectToSender(videoEl: HTMLVideoElement) {
    console.log('CONNECTING TO SENDER');

    const pc = new RTCPeerConnection({})
    pc.ontrack = event => {
        videoEl.srcObject = event.streams[0];
    }

    pc.onicecandidate = async event => {
        if (event.candidate === null) {
            // console.log('SENDER OFFER', pc.localDescription?.sdp);

            const response = await fetch('/webRTCFeed', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(pc.localDescription)
            });

            const answer = await response.json();
            // console.log('SENDER ANSWER', answer.sdp);

            try {
                await pc.setRemoteDescription(new RTCSessionDescription(answer))
            } catch (e) {
                alert(e)
            }

        } else {
            // console.log('SENDER CANDIDATE', event.candidate && event.candidate.candidate)
        }
    }
    pc.addTransceiver('video', {
        'direction': 'recvonly'
    })
    pc.addTransceiver('audio', {
        'direction': 'recvonly'
    })

    const responseDesc = await pc.createOffer()
    pc.setLocalDescription(responseDesc);
}
