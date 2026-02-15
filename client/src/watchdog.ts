export class Watchdog {
    private timeout: number | undefined = undefined;
    private timeoutMs: number;
    private onTimeout: () => void;
    private onReset: () => void;

    constructor(timeoutMs: number, onTimeout: () => void, onReset: () => void) {
        this.timeoutMs = timeoutMs;
        this.onTimeout = onTimeout;
        this.onReset = onReset;
    }

    start(): void {
        this.reset();
    }

    reset(): void {
        this.onReset();

        // Clear existing timeout
        if (this.timeout) {
            clearTimeout(this.timeout);
        }

        // Set new timeout
        this.timeout = setTimeout(() => {
            this.onTimeout();
        }, this.timeoutMs);
    }

    stop(): void {
        if (this.timeout) {
            clearTimeout(this.timeout);
            this.timeout = undefined;
        }
    }
}
