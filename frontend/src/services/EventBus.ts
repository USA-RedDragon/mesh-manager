import mitt from 'mitt'
import type { App } from 'vue'

// See internal/events/events.go
declare type EventType =
  | 'tunnel_disconnection'
  | 'tunnel_connection'
  | 'tunnel_stats'
  | 'total_bandwidth'
  | 'total_traffic'

// See internal/server/api/apimodels/websocket.go
export declare type TunnelStatsEvent = {
  id: number
  rx_bytes_per_sec: number
  tx_bytes_per_sec: number
  rx_bytes: number
  tx_bytes: number
  total_rx_mb: number
  total_tx_mb: number
}

export declare type TunnelConnectionEvent = {
  id: number
  client: string
  connection_time: string
}

export declare type TunnelDisconnectionEvent = {
  id: number
  client: string
}

export declare type TotalBandwidthEvent = {
  RX: number
  TX: number
}

export declare type TotalTrafficEvent = {
  RX: number
  TX: number
}

declare type EventPayloads = {
  tunnel_disconnection: TunnelDisconnectionEvent
  tunnel_connection: TunnelConnectionEvent
  tunnel_stats: TunnelStatsEvent
  total_bandwidth: TotalBandwidthEvent
  total_traffic: TotalTrafficEvent
}

export interface EventBus {
  on<T extends EventType>(type: T, handler: (event: EventPayloads[T]) => void): void
  off<T extends EventType>(type: T, handler?: (event: EventPayloads[T]) => void): void
  emit<T extends EventType>(type: T, event: EventPayloads[T]): void
}

declare module '@vue/runtime-core' {
  interface ComponentCustomProperties {
    $EventBus: EventBus
  }
}

export default {
  install: (app: App<Element>) => {
    app.config.globalProperties.$EventBus = mitt()
  },
}
