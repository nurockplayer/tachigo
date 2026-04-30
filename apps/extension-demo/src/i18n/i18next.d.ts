import 'i18next'
import { defaultNS } from './resources'
import type { Resources } from './resources'

declare module 'i18next' {
  interface CustomTypeOptions {
    defaultNS: typeof defaultNS
    resources: Resources
  }
}
