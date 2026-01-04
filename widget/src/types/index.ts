export interface LocaleTexts {
  consent: {
    title: string;
    body: string;
    accept: string;
    decline: string;
  };
  camera: {
    title: string;
    instruction: string;
    positioning: string;
    capturing: string;
  };
  processing: {
    title: string;
    message: string;
  };
  result: {
    successTitle: string;
    successMessage: string;
    errorTitle: string;
    errorMessage: string;
    retry: string;
    close: string;
  };
  errors: {
    cameraPermission: string;
    cameraNotFound: string;
    cameraError: string;
    networkError: string;
    verificationFailed: string;
    registrationFailed: string;
  };
}

export interface RekkoTheme {
  primaryColor?: string;
  primaryHoverColor?: string;
  backgroundColor?: string;
  surfaceColor?: string;
  textColor?: string;
  textSecondaryColor?: string;
  borderColor?: string;
  borderRadius?: string;
  fontFamily?: string;
}

export interface RekkoConfig {
  publicKey: string;
  locale: string;
  texts: Record<string, LocaleTexts>;
  theme?: RekkoTheme;
  logo?: string;
  apiUrl?: string;
}

export type RekkoMode = 'register' | 'verify';

export interface RekkoResult {
  mode: RekkoMode;
  faceId?: string;
  externalId?: string;
  verified?: boolean;
  confidence?: number;
  registered?: boolean;
}

export type RekkoErrorCode =
  | 'CAMERA_DENIED'
  | 'CAMERA_NOT_FOUND'
  | 'CAMERA_ERROR'
  | 'CAMERA_IN_USE'
  | 'DOMAIN_NOT_ALLOWED'
  | 'INVALID_PUBLIC_KEY'
  | 'SESSION_EXPIRED'
  | 'NETWORK_ERROR'
  | 'VERIFICATION_FAILED'
  | 'REGISTRATION_FAILED'
  | 'FACE_NOT_FOUND'
  | 'NO_FACE_DETECTED'
  | 'MULTIPLE_FACES'
  | 'LOW_QUALITY'
  | 'LIVENESS_FAILED'
  | 'UNKNOWN_ERROR';

export interface RekkoError {
  code: RekkoErrorCode;
  message: string;
  details?: unknown;
}

export type RekkoEventType =
  | 'widget_opened'
  | 'widget_closed'
  | 'consent_accepted'
  | 'consent_declined'
  | 'camera_ready'
  | 'camera_error'
  | 'face_detected'
  | 'face_lost'
  | 'capture_started'
  | 'processing'
  | 'verification_success'
  | 'verification_failed'
  | 'registration_success'
  | 'registration_failed';

export interface RekkoEvent {
  type: RekkoEventType;
  timestamp: number;
  data?: unknown;
}

export interface RekkoOpenOptions {
  mode: RekkoMode;
  externalId?: string;
  onSuccess: (result: RekkoResult) => void;
  onError: (error: RekkoError) => void;
  onEvent?: (event: RekkoEvent) => void;
}

export interface RekkoInstance {
  init: (config: RekkoConfig) => void;
  open: (options: RekkoOpenOptions) => void;
  close: () => void;
  isInitialized: () => boolean;
}

declare global {
  interface Window {
    Rekko: RekkoInstance;
  }
}
