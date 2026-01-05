/**
 * Face Detection Configuration
 * Thresholds and settings for face-api.js detection
 */

export interface DetectionConfig {
  // Face size constraints (percentage of image area)
  minFaceSize: number;
  maxFaceSize: number;

  // Centering tolerance (percentage deviation from center)
  maxCenterOffset: number;

  // Quality thresholds
  minDetectionScore: number;

  // Timing
  stabilityTimeMs: number;
  countdownSeconds: number;

  // FPS settings
  highEndFps: number;
  lowEndFps: number;
  deviceMemoryThreshold: number; // GB - below this, use low-end FPS

  // Liveness (for future phases)
  livenessTimeoutMs: number;
  maxLivenessAttempts: number;

  // Retry
  maxCaptureRetries: number;
  retryDelayMs: number;

  // Model loading
  modelLoadTimeoutMs: number;
}

export const DEFAULT_DETECTION_CONFIG: DetectionConfig = {
  // Face must be between 15% and 65% of image area
  minFaceSize: 0.15,
  maxFaceSize: 0.65,

  // Allow 15% deviation from center
  maxCenterOffset: 0.15,

  // Require 90% confidence in detection
  minDetectionScore: 0.9,

  // Wait 500ms of stable face before countdown
  stabilityTimeMs: 500,

  // 3 second countdown
  countdownSeconds: 3,

  // Adaptive FPS based on device capability
  highEndFps: 30,
  lowEndFps: 15,
  deviceMemoryThreshold: 4, // Devices with < 4GB RAM use low-end FPS

  // Liveness challenge settings
  livenessTimeoutMs: 15000,
  maxLivenessAttempts: 3,

  // Retry settings
  maxCaptureRetries: 5,
  retryDelayMs: 1000,

  // Model loading timeout
  modelLoadTimeoutMs: 10000,
};

/**
 * Detection states for visual feedback
 */
export type DetectionState =
  | 'initializing'      // Loading models
  | 'no_face'           // No face detected
  | 'face_too_small'    // Face too far from camera
  | 'face_too_large'    // Face too close to camera
  | 'face_not_centered' // Face not centered in frame
  | 'poor_lighting'     // Poor image quality/lighting
  | 'multiple_faces'    // More than one face detected
  | 'ready'             // Face properly positioned
  | 'countdown'         // Counting down to capture
  | 'capturing'         // Taking photo
  | 'error';            // Error state

/**
 * State colors for visual feedback
 */
export const STATE_COLORS: Record<DetectionState, string> = {
  initializing: '#6B7280',      // Gray
  no_face: '#6B7280',           // Gray
  face_too_small: '#F59E0B',    // Yellow/Amber
  face_too_large: '#F59E0B',    // Yellow/Amber
  face_not_centered: '#F59E0B', // Yellow/Amber
  poor_lighting: '#F59E0B',     // Yellow/Amber
  multiple_faces: '#EF4444',    // Red
  ready: '#10B981',             // Green
  countdown: '#10B981',         // Green (pulsing)
  capturing: '#10B981',         // Green
  error: '#EF4444',             // Red
};

/**
 * State messages for user feedback
 */
export const STATE_MESSAGES: Record<DetectionState, string> = {
  initializing: 'Carregando...',
  no_face: 'Posicione seu rosto no centro',
  face_too_small: 'Aproxime-se um pouco',
  face_too_large: 'Afaste-se um pouco',
  face_not_centered: 'Centralize seu rosto',
  poor_lighting: 'Melhore a iluminação',
  multiple_faces: 'Apenas um rosto por vez',
  ready: 'Perfeito! Mantenha a posição',
  countdown: '', // Will show countdown number
  capturing: 'Capturando...',
  error: 'Erro na detecção',
};

/**
 * CDN configuration for face-api.js models
 */
export const MODEL_CONFIG = {
  // Using jsDelivr CDN for @vladmandic/face-api models
  cdnBaseUrl: 'https://cdn.jsdelivr.net/npm/@vladmandic/face-api@1.7.15/model',

  // Models needed for Phase 1 (basic detection + landmarks)
  phase1Models: [
    'tiny_face_detector',
    'face_landmark_68_tiny',
  ],

  // Additional models for Phase 5 (liveness)
  livenessModels: [
    'face_expression',
  ],

  // Fallback to local models if CDN fails
  localFallbackPath: '/models',
};

/**
 * Get optimal FPS based on device capability
 */
export function getOptimalFps(config: DetectionConfig = DEFAULT_DETECTION_CONFIG): number {
  // Check if navigator.deviceMemory is available
  const deviceMemory = (navigator as Navigator & { deviceMemory?: number }).deviceMemory;

  if (deviceMemory !== undefined && deviceMemory < config.deviceMemoryThreshold) {
    return config.lowEndFps;
  }

  return config.highEndFps;
}

/**
 * Calculate detection interval in ms from FPS
 */
export function getDetectionIntervalMs(fps: number): number {
  return Math.round(1000 / fps);
}
