/**
 * Face Detection Types
 * Types for face-api.js integration and detection results
 */

import type { DetectionState } from '../config/detection';

/**
 * 2D Point coordinates
 */
export interface Point {
  x: number;
  y: number;
}

/**
 * Bounding box for detected face
 */
export interface FaceBoundingBox {
  x: number;
  y: number;
  width: number;
  height: number;
}

/**
 * Face position relative to frame
 */
export interface FacePosition {
  /** Bounding box of detected face */
  boundingBox: FaceBoundingBox;

  /** Center point of face */
  center: Point;

  /** Face size as percentage of image area (0-1) */
  sizeRatio: number;

  /** Distance from frame center as percentage (0-1) */
  centerOffset: number;

  /** Is face within acceptable size range */
  isSizeValid: boolean;

  /** Is face properly centered */
  isCentered: boolean;
}

/**
 * Key facial landmarks
 */
export interface FaceLandmarks {
  /** Left eye center point */
  leftEye: Point;

  /** Right eye center point */
  rightEye: Point;

  /** Nose tip point */
  nose: Point;

  /** Mouth center point */
  mouth: Point;

  /** Left mouth corner */
  leftMouth: Point;

  /** Right mouth corner */
  rightMouth: Point;

  /** Jaw outline points */
  jawOutline: Point[];

  /** All 68 landmark points (raw) */
  raw: Point[];
}

/**
 * Detection quality metrics
 */
export interface DetectionQuality {
  /** Overall detection confidence score (0-1) */
  score: number;

  /** Estimated lighting quality (0-1) */
  lightingScore: number;

  /** Face blur estimation (0-1, higher = less blur) */
  sharpnessScore: number;

  /** Are eyes open and visible */
  eyesVisible: boolean;

  /** Is mouth visible */
  mouthVisible: boolean;

  /** Overall quality assessment */
  isAcceptable: boolean;
}

/**
 * Complete face detection result
 */
export interface FaceDetectionResult {
  /** Is a face detected */
  detected: boolean;

  /** Number of faces detected */
  faceCount: number;

  /** Position information (if detected) */
  position: FacePosition | null;

  /** Landmark points (if detected) */
  landmarks: FaceLandmarks | null;

  /** Quality metrics (if detected) */
  quality: DetectionQuality | null;

  /** Raw detection score from face-api.js */
  rawScore: number;

  /** Timestamp of detection */
  timestamp: number;
}

/**
 * Face detection hook state
 */
export interface FaceDetectionState {
  /** Current detection state for UI */
  state: DetectionState;

  /** Latest detection result */
  detection: FaceDetectionResult | null;

  /** Are models loaded and ready */
  isReady: boolean;

  /** Is detection currently running */
  isDetecting: boolean;

  /** Has loading/detection failed */
  hasError: boolean;

  /** Error message if any */
  errorMessage: string | null;

  /** Model loading progress (0-100) */
  loadingProgress: number;

  /** Time face has been stable in ms */
  stabilityTime: number;

  /** Current countdown value (null if not counting) */
  countdown: number | null;
}

/**
 * Face detection hook options
 */
export interface FaceDetectionOptions {
  /** Video element to analyze */
  videoRef: import('preact').RefObject<HTMLVideoElement>;

  /** Enable detection on mount */
  autoStart?: boolean;

  /** Custom detection config overrides */
  configOverrides?: Partial<import('../config/detection').DetectionConfig>;

  /** Callback when face becomes ready */
  onFaceReady?: () => void;

  /** Callback when detection state changes */
  onStateChange?: (state: DetectionState) => void;

  /** Callback when countdown completes */
  onCountdownComplete?: () => void;

  /** Callback on error */
  onError?: (error: Error) => void;
}

/**
 * Face detection hook return type
 */
export interface UseFaceDetectionReturn {
  /** Current state */
  state: FaceDetectionState;

  /** Start detection */
  start: () => Promise<void>;

  /** Stop detection */
  stop: () => void;

  /** Reset state (for retry) */
  reset: () => void;

  /** Start countdown manually */
  startCountdown: () => void;

  /** Cancel countdown */
  cancelCountdown: () => void;

  /** Get current detection result */
  getDetection: () => FaceDetectionResult | null;
}

/**
 * Model loading state
 */
export interface ModelLoadingState {
  /** Is currently loading */
  isLoading: boolean;

  /** Has completed loading */
  isLoaded: boolean;

  /** Loading error if any */
  error: Error | null;

  /** Progress percentage (0-100) */
  progress: number;

  /** Which models are loaded */
  loadedModels: string[];
}

/**
 * Face-api.js model types we use
 */
export type FaceApiModel =
  | 'tiny_face_detector'
  | 'face_landmark_68_tiny'
  | 'face_expression';

/**
 * Detection event types for callbacks
 */
export type DetectionEventType =
  | 'models_loading'
  | 'models_loaded'
  | 'models_error'
  | 'detection_started'
  | 'detection_stopped'
  | 'face_detected'
  | 'face_lost'
  | 'face_ready'
  | 'countdown_started'
  | 'countdown_tick'
  | 'countdown_complete'
  | 'countdown_cancelled'
  | 'error';

/**
 * Detection event payload
 */
export interface DetectionEvent {
  type: DetectionEventType;
  timestamp: number;
  data?: unknown;
}
