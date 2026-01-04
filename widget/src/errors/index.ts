import type { RekkoError, RekkoErrorCode } from '@/types';

export function createError(code: RekkoErrorCode, message: string, details?: unknown): RekkoError {
  return { code, message, details };
}

export const Errors = {
  cameraDenied: (): RekkoError =>
    createError('CAMERA_DENIED', 'Camera access was denied by the user'),

  cameraNotFound: (): RekkoError =>
    createError('CAMERA_NOT_FOUND', 'No camera device was found'),

  cameraError: (details?: unknown): RekkoError =>
    createError('CAMERA_ERROR', 'Failed to access camera', details),

  cameraInUse: (): RekkoError =>
    createError('CAMERA_IN_USE', 'Camera is being used by another application'),

  domainNotAllowed: (): RekkoError =>
    createError('DOMAIN_NOT_ALLOWED', 'This domain is not authorized to use the widget'),

  invalidPublicKey: (): RekkoError =>
    createError('INVALID_PUBLIC_KEY', 'The provided public key is invalid'),

  sessionExpired: (): RekkoError =>
    createError('SESSION_EXPIRED', 'The widget session has expired'),

  networkError: (details?: unknown): RekkoError =>
    createError('NETWORK_ERROR', 'A network error occurred', details),

  verificationFailed: (details?: unknown): RekkoError =>
    createError('VERIFICATION_FAILED', 'Face verification failed', details),

  registrationFailed: (details?: unknown): RekkoError =>
    createError('REGISTRATION_FAILED', 'Face registration failed', details),

  faceNotFound: (): RekkoError =>
    createError('FACE_NOT_FOUND', 'No registered face found for this ID'),

  noFaceDetected: (): RekkoError =>
    createError('NO_FACE_DETECTED', 'No face was detected in the image'),

  multipleFaces: (): RekkoError =>
    createError('MULTIPLE_FACES', 'Multiple faces were detected, only one is allowed'),

  lowQuality: (): RekkoError =>
    createError('LOW_QUALITY', 'Image quality is too low for processing'),

  livenessFailed: (): RekkoError =>
    createError('LIVENESS_FAILED', 'Liveness check failed'),

  unknown: (details?: unknown): RekkoError =>
    createError('UNKNOWN_ERROR', 'An unknown error occurred', details),
};
