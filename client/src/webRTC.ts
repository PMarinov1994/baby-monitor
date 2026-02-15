
export async function connectToSender(videoEl: HTMLVideoElement) {
    const iceServerReq = await fetch(`http://${window.location.hostname}:8080/cam/whep`, {
        method: 'OPTIONS',
        headers: {
        },
    })

    const link = iceServerReq.headers.get('Link')
    const iceServers = linkToIceServers(link!)

    const pc = new RTCPeerConnection({
        iceServers,
    })

    pc.ontrack = event => {
        if (videoEl.srcObject !== event.streams[0]) {
            videoEl.srcObject = event.streams[0]
        }
    }

    const queueCandidate: RTCIceCandidate[] = []
    pc.onicecandidate = event => {
        if (event.candidate !== null)
            queueCandidate.push(event.candidate)
    }

    const audioTRansceiver = pc.addTransceiver('audio', { direction: 'recvonly' })
    audioTRansceiver.receiver.jitterBufferTarget = 1

    const videoTransceiver = pc.addTransceiver('video', { direction: 'recvonly' })
    videoTransceiver.receiver.jitterBufferTarget = 1

    try {
        const offer = await pc.createOffer()
        await pc.setLocalDescription(offer);

        if (pc.localDescription === null) {
            alert("pc.localDescription is null")
            return
        }

        const response = await fetch(`http://${window.location.hostname}:8080/cam/whep`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/sdp'
            },
            body: offer.sdp
        });

        if (!response.ok) {
            throw new Error('Failed to send offer');
        }

        const answer = await response.text();

        await pc.setRemoteDescription(new RTCSessionDescription(
            {
                type: 'answer',
                sdp: answer,
            }));

    } catch (e) {
        alert(e)
    }
}


function linkToIceServers(links: string) {
    return (links !== null) ? links.split(', ').map((link) => {
        const m = link.match(/^<(.+?)>; rel="ice-server"(; username="(.*?)"; credential="(.*?)"; credential-type="password")?/i);

        const ret = {
            urls: [m![1]],
        };

        return ret;
    }) : [];
}
