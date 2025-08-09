let ws: WebSocket | null = null;

const DATA_SEPARATOR = "&&&"

const REQ_SOUND_CARDS = "getSoundCards"
const RES_SOUND_CARDS = "gotSoundCards"

const REQ_CHANGE_SOUND = "setSound"
const RES_CHANGE_SOUND = "gotSound"

const soundCardSelect = document.getElementById('soundCards') as HTMLSelectElement
const outputsSelect = document.getElementById('outputs') as HTMLSelectElement
const volumeSlider = document.getElementById('volume') as HTMLInputElement


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

let soundCards: SoundCard[] = []

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
    const cardIndex = parseInt(soundCardSelect.value, 10)
    const channelndex = parseInt(outputsSelect.value, 10)

    const card = soundCards[cardIndex]
    const channel = card.outChannels[channelndex]

    const value = parseInt(volumeSlider.value, 10)
    console.log(`${card.shortName} - ${channel.name} -> ${value}`)

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

export function wsConnect(): void {
    const loc = window.location;

    ws = new WebSocket(`ws://${loc.host}/api`);

    ws.onopen = () => {
        console.log('WebSocket connected');
        ws?.send(REQ_SOUND_CARDS)
    };

    ws.onmessage = (event: MessageEvent) => {
        console.log('WebSocket message:', event.data);
        const response: string = event.data as string

        const parts = response.split(DATA_SEPARATOR)
        switch (parts[0]) {
            case RES_SOUND_CARDS:
                soundCards = JSON.parse(parts[1])
                console.log(soundCards)

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
        }
    };

    ws.onerror = (error: Event) => {
        console.error('WebSocket error:', error);
    };

    ws.onclose = () => {
        console.log('WebSocket connection closed');
    };
}
