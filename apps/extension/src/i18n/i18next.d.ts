import 'i18next'
import type { defaultNS, Resources } from './resources'

declare module 'i18next' {
  interface CustomTypeOptions {
    defaultNS: typeof defaultNS
    resources: Resources
  }
}
