import VueRouter, { Route } from 'vue-router'

interface ToastOptions {
  summary: string
  detail: string
  life: number
  severity: 'success' | 'info' | 'warn' | 'error'
}

interface ToastServiceMethods {
  add(opts: ToastOptions): void
}

interface ConfirmOptions {
  message: string
  header: string
  icon: string
  acceptClass: string
  accept: () => void
  reject: () => void
}

interface ConfirmServiceMethods {
  require(opts: ConfirmOptions): void
}

declare module '@vue/runtime-core' {
  interface ComponentCustomProperties {
    $toast: ToastServiceMethods
    $confirm: ConfirmServiceMethods
    $router: VueRouter
    $route: Route
  }
}
