import { isMock, type ApiClient } from './client'
import { httpApi } from './http'
import { mockApi } from './mock'

export type { ApiClient } from './client'
export { isMock } from './client'

export const api: ApiClient = isMock() ? mockApi : httpApi
