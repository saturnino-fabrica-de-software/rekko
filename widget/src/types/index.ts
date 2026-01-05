/**
 * Rekko Widget Types
 * Enterprise-grade type definitions for bilheteria platform
 */

// ============================================================================
// LOCALE & TEXTS
// ============================================================================

export interface LocaleTexts {
  consent: {
    title: string;
    body: string;
    accept: string;
    decline: string;
    privacyPolicy?: string;
    dataRetention?: string;
  };
  orientation: {
    title: string;
    subtitle: string;
    instructions: {
      neutral: string;
      visible: string;
      lighting: string;
      framing: string;
    };
    continue: string;
  };
  camera: {
    title: string;
    instruction: string;
    positioning: string;
    capturing: string;
  };
  liveness: {
    title: string;
    challenges: {
      turn_left: string;
      turn_right: string;
      blink: string;
    };
    success: string;
    failed: string;
    timeout: string;
    retry: string;
    skip: string;
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

// ============================================================================
// THEME & STYLING
// ============================================================================

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
  // Extended theme options
  accentColor?: string;
  successColor?: string;
  errorColor?: string;
}

// ============================================================================
// EVENT CONTEXT (Bilheteria-specific)
// ============================================================================

export interface EventContext {
  /** Event name to display */
  name: string;
  /** Event date (ISO string or formatted) */
  date?: string;
  /** Event location/venue */
  venue?: string;
  /** Entry gate (e.g., "VIP Norte", "Portão A") */
  entryGate?: string;
  /** Ticket type (e.g., "Premium", "Pista") */
  ticketType?: string;
  /** Event logo URL */
  logo?: string;
  /** Organizer name */
  organizer?: string;
}

// ============================================================================
// CONSENT (LGPD Compliance)
// ============================================================================

export interface ConsentConfig {
  /** Required consents that must be accepted */
  required: ConsentType[];
  /** Optional consents */
  optional?: ConsentType[];
  /** Privacy policy URL (required for LGPD) */
  privacyPolicyUrl: string;
  /** Terms of service URL */
  termsUrl?: string;
  /** Data retention period in hours (default: 24) */
  retentionHours?: number;
}

export type ConsentType =
  | 'identity_verification'    // Required: facial verification
  | 'biometric_processing'     // Required: biometric data processing
  | 'post_event_storage'       // Optional: store data after event
  | 'marketing_communications' // Optional: receive marketing
  | 'analytics';               // Optional: anonymous analytics

export interface ConsentRecord {
  type: ConsentType;
  granted: boolean;
  timestamp: number;
  version: string;
}

// ============================================================================
// CONFIGURATION
// ============================================================================

export interface RekkoConfig {
  publicKey: string;
  locale: string;
  texts: Record<string, LocaleTexts>;
  theme?: RekkoTheme;
  logo?: string;
  apiUrl?: string;
  /** Enable debug mode */
  debug?: boolean;
}

export type RekkoMode = 'register' | 'verify';

// ============================================================================
// RESULTS & ERRORS
// ============================================================================

export interface RekkoResult {
  mode: RekkoMode;
  faceId?: string;
  externalId?: string;
  verified?: boolean;
  confidence?: number;
  registered?: boolean;
  /** Consent records for audit trail */
  consents?: ConsentRecord[];
  /** Processing latency in ms */
  latencyMs?: number;
}

export type RekkoErrorCode =
  | 'CAMERA_DENIED'
  | 'CAMERA_NOT_FOUND'
  | 'CAMERA_ERROR'
  | 'CAMERA_IN_USE'
  | 'DOMAIN_NOT_ALLOWED'
  | 'ORIGIN_NOT_ALLOWED'
  | 'INVALID_PUBLIC_KEY'
  | 'SESSION_EXPIRED'
  | 'SESSION_NOT_FOUND'
  | 'NETWORK_ERROR'
  | 'VERIFICATION_FAILED'
  | 'REGISTRATION_FAILED'
  | 'FACE_NOT_FOUND'
  | 'FACE_ALREADY_EXISTS'
  | 'NO_FACE_DETECTED'
  | 'MULTIPLE_FACES'
  | 'LOW_QUALITY'
  | 'LIVENESS_FAILED'
  | 'CONSENT_REQUIRED'
  | 'UNKNOWN_ERROR';

export interface RekkoError {
  code: RekkoErrorCode;
  message: string;
  details?: unknown;
  /** Suggested action for user */
  action?: 'retry' | 'contact_support' | 'check_camera' | 'accept_consent';
}

// ============================================================================
// EVENTS
// ============================================================================

export type RekkoEventType =
  | 'widget_opened'
  | 'widget_closed'
  | 'consent_accepted'
  | 'consent_declined'
  | 'consent_partial'        // Some optional consents declined
  | 'camera_ready'
  | 'camera_error'
  | 'face_detected'
  | 'face_lost'
  | 'face_detection_state'
  | 'liveness_started'
  | 'liveness_challenge'
  | 'liveness_success'
  | 'liveness_failed'
  | 'capture_started'
  | 'processing'
  | 'verification_success'
  | 'verification_failed'
  | 'registration_success'
  | 'registration_failed'
  | 'network_status_changed'; // Connection quality changed

export interface RekkoEvent {
  type: RekkoEventType;
  timestamp: number;
  data?: unknown;
}

// ============================================================================
// OPEN OPTIONS (Extended for bilheteria)
// ============================================================================

export interface RekkoOpenOptions {
  mode: RekkoMode;
  externalId?: string;
  onSuccess: (result: RekkoResult) => void;
  onError: (error: RekkoError) => void;
  onEvent?: (event: RekkoEvent) => void;
  /** Event context for bilheteria (shows event info in UI) */
  eventContext?: EventContext;
  /** Consent configuration for LGPD compliance */
  consent?: ConsentConfig;
  /** Skip consent screen if already collected */
  skipConsent?: boolean;
  /** Custom metadata to attach to result */
  metadata?: Record<string, unknown>;
}

// ============================================================================
// INSTANCE
// ============================================================================

export interface RekkoInstance {
  init: (config: RekkoConfig) => void;
  open: (options: RekkoOpenOptions) => void;
  close: () => void;
  isInitialized: () => boolean;
  /** Get current version */
  version: () => string;
}

declare global {
  interface Window {
    Rekko: RekkoInstance;
  }
}

// Re-export face detection types
export * from './faceDetection';
