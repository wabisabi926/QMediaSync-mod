export type V115AuthStatus = 'idle' | 'waiting' | 'scanned' | 'confirmed' | 'expired' | 'failed'

export interface V115QrCodePayload {
  uid: string
  time: number
  sign: string
  qrcode: string
  expires: number
}

export interface V115QrCodeStatusPayload {
  status: V115AuthStatus
  tip: string
}
