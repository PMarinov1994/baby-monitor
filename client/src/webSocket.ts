let ws: WebSocket | null = null;

const DATA_SEPARATOR = "&&&"

const REQ_TOGGLE_NIGHT_VISION = "setToggleNightVision"
const RES_TOGGLE_NIGHT_VISION = "gotToggleNightVision"

const REQ_UPDATE_STATE = "setUpdateState"
// const RES_UPDATE_STATE = "gotUpdateState"

const REQ_GET_STATE = "getGetState"
const RES_GET_STATE = "gotGetState"

interface StateUpdate {
    nightVision: boolean;
}

const toggleNightVision = document.getElementById('toggleNightVision') as HTMLInputElement

let ignoreEvent: boolean = false


function updateState(state: StateUpdate) {
    toggleNightVision.checked = state.nightVision

    ignoreEvent = true
    toggleNightVision.dispatchEvent(new Event('change'))
    ignoreEvent = false
}


toggleNightVision.addEventListener('change', () => {
    if (ws === null || ignoreEvent)
        return

    const chunks: string[] = [
        REQ_TOGGLE_NIGHT_VISION,
        toggleNightVision.checked.toString(),
    ]

    ws.send(chunks.join(DATA_SEPARATOR))
})


export function wsConnect(): void {
    const loc = window.location;

    ws = new WebSocket(`ws://${loc.host}/api`);

    ws.onopen = () => {
        console.log('WebSocket connected');
        ws?.send(REQ_GET_STATE)
    };

    ws.onmessage = (event: MessageEvent) => {
        // console.log('WebSocket message:', event.data);
        const response: string = event.data as string

        const parts = response.split(DATA_SEPARATOR)
        switch (parts[0]) {
            case REQ_UPDATE_STATE:
                const state = JSON.parse(parts[1]) as StateUpdate
                updateState(state)
                break

            case RES_TOGGLE_NIGHT_VISION:
            case RES_GET_STATE:
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
