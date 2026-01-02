export declare type WebsocketMessageHandler = (event: MessageEvent) => void

const initialReconnectDelay = 300
const maxReconnectDelay = 15000

export class Websocket {
  url: string
  timeout: number
  socket: WebSocket | null
  onMessage: WebsocketMessageHandler
  currentReconnectDelay: number

  /**
   * @param {string} url - The URL of the WebSocket server.
   * @param {function} onMessage - Callback function to handle incoming messages.
   */
  constructor(url: string, onMessage: WebsocketMessageHandler) {
    this.url = url
    this.timeout = 3000
    this.socket = null
    this.onMessage = onMessage
    this.currentReconnectDelay = initialReconnectDelay
  }

  connect() {
    this.socket = new WebSocket(this.url)
    this.mapSocketEvents()
  }

  close() {
    if (this.socket) {
      this.socket.close()
      this.socket = null
    }
  }

  onWebsocketOpen() {
    console.log('Connected to websocket')
    this.currentReconnectDelay = initialReconnectDelay
    this.socket?.send('PING')
  }

  onWebsocketError() {
    console.log('Disconnected from websocket')
    this.socket = null
    this.reconnectToWebsocket()
  }

  reconnectToWebsocket() {
    setTimeout(
      () => {
        if (this.currentReconnectDelay < maxReconnectDelay) {
          this.currentReconnectDelay *= 2
        }
        this.connect()
      },
      this.currentReconnectDelay + Math.floor(Math.random() * 1000),
    )
  }

  mapSocketEvents() {
    this.socket?.addEventListener('open', this.onWebsocketOpen.bind(this))
    this.socket?.addEventListener('error', this.onWebsocketError.bind(this))

    this.socket?.addEventListener('message', (event) => {
      if (event.data == 'PONG') {
        setTimeout(() => {
          this.socket?.send('PING')
        }, 1000)
        return
      } else {
        this.onMessage(event)
      }
    })
  }
}
