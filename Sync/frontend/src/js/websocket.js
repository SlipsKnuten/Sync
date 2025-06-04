class WebSocketManager {
    constructor(userId, sessionCode, onMessage, onStatusChange) {
        this.userId = userId;
        this.sessionCode = sessionCode;
        this.onMessage = onMessage;
        this.onStatusChange = onStatusChange;
        this.ws = null;
        this.reconnectTimeout = null;
    }

    connect() {
        // Use relative URL to work with any host
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.hostname}:8080/ws?userId=${this.userId}&session=${this.sessionCode}`;
        this.ws = new WebSocket(wsUrl);
        
        this.ws.onopen = () => {
            if (this.onStatusChange) {
                this.onStatusChange('connected');
            }
            if (this.reconnectTimeout) {
                clearTimeout(this.reconnectTimeout);
                this.reconnectTimeout = null;
            }
        };

        this.ws.onmessage = (event) => {
            const msg = JSON.parse(event.data);
            if (this.onMessage) {
                this.onMessage(msg);
            }
        };

        this.ws.onclose = () => {
            if (this.onStatusChange) {
                this.onStatusChange('disconnected');
            }
            this.scheduleReconnect();
        };

        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };
    }

    scheduleReconnect() {
        if (!this.reconnectTimeout) {
            this.reconnectTimeout = setTimeout(() => {
                this.connect();
            }, 3000);
        }
    }

    sendMessage(type, data) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify({ type, ...data }));
        }
    }

    close() {
        if (this.reconnectTimeout) {
            clearTimeout(this.reconnectTimeout);
        }
        if (this.ws) {
            this.ws.close();
        }
    }
}