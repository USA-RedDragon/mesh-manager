declare module 'cypress-mochawesome-reporter/plugin' {
  const plugin: (on: Cypress.PluginEvents, config?: Cypress.PluginConfigOptions) => void
  export default plugin
}
