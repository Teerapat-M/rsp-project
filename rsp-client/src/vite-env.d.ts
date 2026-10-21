/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_USE_MOCK?: string
  readonly VITE_API_BASE?: string
  readonly VITE_WS_URL?: string
  readonly VITE_GOOGLE_CLIENT_ID?: string
}

interface GoogleIdentity {
  accounts: {
    id: {
      initialize: (cfg: { client_id: string; callback: (r: { credential?: string }) => void }) => void
      prompt: () => void
    }
  }
}

export {}

declare global {
  interface Window {
    google?: GoogleIdentity
  }
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
