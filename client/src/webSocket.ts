let ws: WebSocket | null = null;

export function wsConnect(): void {
    ws = new WebSocket('ws://192.168.200.109:8080/api');

    ws.onopen = () => {
        console.log('WebSocket connected');
    };

    ws.onmessage = (event: MessageEvent) => {
        console.log('WebSocket message:', event.data);
    };

    ws.onerror = (error: Event) => {
        console.error('WebSocket error:', error);
    };

    ws.onclose = () => {
        console.log('WebSocket connection closed');
    };
}
