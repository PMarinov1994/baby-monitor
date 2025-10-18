let ws: WebSocket | null = null;

const DATA_SEPARATOR = "&&&"

const REQ_SOUND_CARDS = "getSoundCards"
const RES_SOUND_CARDS = "gotSoundCards"

const REQ_CHANGE_SOUND = "setSound"
const RES_CHANGE_SOUND = "gotSound"

const REQ_WS_ID = "getWsId"
const RES_WS_ID = "setWsId"

const REQ_TOGGLE_NIGHT_VISION = "setToggleNightVision"
const RES_TOGGLE_NIGHT_VISION = "gotToggleNightVision"

const REQ_TOGGLE_SOUND_DRAW = "setToggleSoundDraw"
const RES_TOGGLE_SOUND_DRAW = "gotToggleSoundDraw"

const REQ_UPDATE_STATE = "setUpdateState"
// const RES_UPDATE_STATE = "gotUpdateState"

const REQ_GET_STATE = "getGetState"
const RES_GET_STATE = "gotGetState"


interface OutputChannel {
    name: string;
    minVolume: number;
    maxVolume: number;
    curVolume: number;
}

interface SoundCard {
    shortName: string;
    longName: string;
    mixerName: string;
    outChannels: OutputChannel[];
}

interface StateUpdate {
    soundCards: SoundCard[];
    nightVision: boolean;
    drawSound: boolean;
}

const soundCardSelect = document.getElementById('soundCards') as HTMLSelectElement
const outputsSelect = document.getElementById('outputs') as HTMLSelectElement
const volumeSlider = document.getElementById('volume') as HTMLInputElement
const toggleNightVision = document.getElementById('toggleNightVision') as HTMLInputElement
const toggleSoundDraw = document.getElementById('toggleSoundWaveDraw') as HTMLInputElement

let soundCards: SoundCard[] = []
let ignoreEvent: boolean = false

function updateState(state: StateUpdate) {
    state.soundCards.forEach((sc, scIdx) => {
        sc.outChannels.forEach((oc, ocIdx) => {
            soundCards[scIdx].outChannels[ocIdx].curVolume = oc.curVolume
        })
    })

    const cardIdx = parseInt(soundCardSelect.value, 10)
    const outputIdx = parseInt(outputsSelect.value, 10)
    const currVol = soundCards[cardIdx].outChannels[outputIdx].curVolume.toString()

    volumeSlider.value = currVol
    toggleNightVision.checked = state.nightVision
    toggleSoundDraw.checked = state.drawSound

    ignoreEvent = true
    volumeSlider.dispatchEvent(new Event('change'))
    toggleNightVision.dispatchEvent(new Event('change'))
    toggleSoundDraw.dispatchEvent(new Event('change'))
    ignoreEvent = false
}


function resetVolumeSlider(): void {
    volumeSlider.min = "0"
    volumeSlider.max = "0"
    volumeSlider.value = "0"
    volumeSlider.disabled = true
}

function updateVolumeSlider(channel: OutputChannel): void {
    volumeSlider.min = channel.minVolume.toString()
    volumeSlider.max = channel.maxVolume.toString()
    volumeSlider.value = channel.curVolume.toString()
    volumeSlider.disabled = false
}

soundCardSelect.addEventListener('change', () => {
    const cardIndex = parseInt(soundCardSelect.value, 10)
    const card = soundCards[cardIndex]

    outputsSelect.innerHTML = ""

    card.outChannels.forEach((channel, index) => {
        const opt = document.createElement('option')
        opt.value = index.toString()
        opt.textContent = channel.name
        outputsSelect.appendChild(opt)
    })

    if (card.outChannels.length > 0) {
        outputsSelect.disabled = false
        updateVolumeSlider(card.outChannels[0])
    } else {
        outputsSelect.disabled = true
        resetVolumeSlider()
    }
})

outputsSelect.addEventListener('change', () => {
    const cardIndex = parseInt(soundCardSelect.value, 10)
    const channelndex = parseInt(outputsSelect.value, 10)
    const channel = soundCards[cardIndex].outChannels[channelndex]

    updateVolumeSlider(channel)
})

volumeSlider.addEventListener('change', () => {
    if (ignoreEvent)
        return

    const cardIndex = parseInt(soundCardSelect.value, 10)
    const channelndex = parseInt(outputsSelect.value, 10)

    const card = soundCards[cardIndex]
    const channel = card.outChannels[channelndex]

    const value = parseInt(volumeSlider.value, 10)
    // console.log(`${card.shortName} - ${channel.name} -> ${value}`)

    if (ws === null)
        return

    const chunks: string[] = [
        REQ_CHANGE_SOUND,
        card.longName,
        channel.name,
        value.toString(),
    ]

    ws.send(chunks.join(DATA_SEPARATOR))
})

toggleNightVision.addEventListener('change', () => {
    if (ws === null || ignoreEvent)
        return

    const chunks: string[] = [
        REQ_TOGGLE_NIGHT_VISION,
        toggleNightVision.checked.toString(),
    ]

    ws.send(chunks.join(DATA_SEPARATOR))
})

toggleSoundDraw.addEventListener('change', () => {
    if (ws === null || ignoreEvent)
        return

    const chunks: string[] = [
        REQ_TOGGLE_SOUND_DRAW,
        toggleSoundDraw.checked.toString(),
    ]

    ws.send(chunks.join(DATA_SEPARATOR))
})

export function wsConnect(pc: RTCPeerConnection, dc: RTCDataChannel): void {
    const loc = window.location;

    ws = new WebSocket(`ws://${loc.host}/api`);

    ws.onopen = () => {
        console.log('WebSocket connected');
        ws?.send(REQ_WS_ID)
        ws?.send(REQ_SOUND_CARDS)
        ws?.send(REQ_GET_STATE)
    };

    ws.onmessage = (event: MessageEvent) => {
        console.log('WebSocket message:', event.data);
        const response: string = event.data as string

        const parts = response.split(DATA_SEPARATOR)
        switch (parts[0]) {
            case RES_SOUND_CARDS:
                soundCards = JSON.parse(parts[1])
                // console.log(soundCards)

                soundCards.forEach((card, index) => {
                    const opt = document.createElement('option')
                    opt.value = index.toString()
                    opt.textContent = card.shortName
                    soundCardSelect.appendChild(opt)
                })

                soundCardSelect.value = "0"
                soundCardSelect.dispatchEvent(new Event('change'))

                break

            case RES_CHANGE_SOUND:
                if (parts[1].toLowerCase().includes("error")) {
                    alert(parts[1])
                }
                break

            case RES_WS_ID:
                const wsId = parts[1]
                console.log("Sending client id over data channel", wsId)

                dc.onmessage = _ => dc.close()
                if (dc.readyState !== "open") {
                    dc.onopen = () => {
                        console.log("On datachannel open")
                        dc.send(wsId)
                    }
                } else {
                    dc.send(wsId)
                }

                break

            case RES_TOGGLE_NIGHT_VISION:
                if (parts[1].toLowerCase().includes("error")) {
                    alert(parts[1])
                }
                break

            case RES_TOGGLE_SOUND_DRAW:
                if (parts[1].toLowerCase().includes("error")) {
                    alert(parts[1])
                }
                break

            case RES_GET_STATE:
                if (parts[1].toLowerCase().includes("error")) {
                    alert(parts[1])
                }
                break

            case REQ_UPDATE_STATE:
                const state = JSON.parse(parts[1]) as StateUpdate
                console.log("Curr state:\n", parts[1])
                updateState(state)
                break
        }
    };

    ws.onerror = (error: Event) => {
        console.error('WebSocket error:', error);
    };

    ws.onclose = () => {
        console.log('WebSocket connection closed');
        pc.close()
    };
}
