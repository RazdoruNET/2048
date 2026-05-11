export interface WSMessage {
  type: string;
  data: any;
  timestamp: string;
}

export interface MLStatusUpdate {
  enabled: boolean;
  last_update: string;
  model_version: string;
}

export interface RequestCompleted {
  request_id: string;
  domain: string;
  success: boolean;
  latency: number;
  techniques: string[];
  dpi_type: string;
  confidence: number;
  completed_at: string;
}

export interface TechniqueEffectivenessChanged {
  technique: string;
  domain: string;
  effectiveness: number;
  sample_count: number;
  updated_at: string;
}

export type WSMessageHandler = (message: WSMessage) => void;

export class WebSocketClient {
  private ws: WebSocket | null = null;
  private handlers: Map<string, WSMessageHandler[]> = new Map();
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;
  private isConnecting = false;
  private url: string;

  constructor(url?: string) {
    this.url = url || 'ws://localhost:8080/api/v1/ws';
  }

  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      if (this.isConnecting) {
        reject(new Error('Connection already in progress'));
        return;
      }

      if (this.ws && this.ws.readyState === WebSocket.OPEN) {
        resolve();
        return;
      }

      this.isConnecting = true;

      try {
        this.ws = new WebSocket(this.url);

        this.ws.onopen = () => {
          console.log('WebSocket connected');
          this.isConnecting = false;
          this.reconnectAttempts = 0;
          resolve();
        };

        this.ws.onmessage = (event) => {
          try {
            const message: WSMessage = JSON.parse(event.data);
            this.handleMessage(message);
          } catch (error) {
            console.error('Failed to parse WebSocket message:', error);
          }
        };

        this.ws.onclose = (event) => {
          console.log('WebSocket disconnected:', event.code, event.reason);
          this.isConnecting = false;
          this.ws = null;
          
          if (!event.wasClean && this.reconnectAttempts < this.maxReconnectAttempts) {
            this.scheduleReconnect();
          }
        };

        this.ws.onerror = (error) => {
          console.error('WebSocket error:', error);
          this.isConnecting = false;
          reject(error);
        };
      } catch (error) {
        this.isConnecting = false;
        reject(error);
      }
    });
  }

  private scheduleReconnect() {
    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
    
    console.log(`Scheduling reconnect attempt ${this.reconnectAttempts} in ${delay}ms`);
    
    setTimeout(() => {
      if (this.reconnectAttempts <= this.maxReconnectAttempts) {
        this.connect().catch(console.error);
      }
    }, delay);
  }

  private handleMessage(message: WSMessage) {
    const handlers = this.handlers.get(message.type) || [];
    handlers.forEach(handler => {
      try {
        handler(message);
      } catch (error) {
        console.error(`Error in handler for message type ${message.type}:`, error);
      }
    });
  }

  on(messageType: string, handler: WSMessageHandler) {
    if (!this.handlers.has(messageType)) {
      this.handlers.set(messageType, []);
    }
    this.handlers.get(messageType)!.push(handler);
  }

  off(messageType: string, handler: WSMessageHandler) {
    const handlers = this.handlers.get(messageType);
    if (handlers) {
      const index = handlers.indexOf(handler);
      if (index > -1) {
        handlers.splice(index, 1);
      }
    }
  }

  disconnect() {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.reconnectAttempts = this.maxReconnectAttempts; // Prevent reconnection
  }

  isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
  }

  // Convenience methods for specific message types
  onMLStatusUpdate(handler: (data: MLStatusUpdate) => void) {
    this.on('ml.status_update', (message) => handler(message.data));
  }

  onRequestCompleted(handler: (data: RequestCompleted) => void) {
    this.on('request.completed', (message) => handler(message.data));
  }

  onTechniqueEffectivenessChanged(handler: (data: TechniqueEffectivenessChanged) => void) {
    this.on('technique.effectiveness_changed', (message) => handler(message.data));
  }
}

export const wsClient = new WebSocketClient();
