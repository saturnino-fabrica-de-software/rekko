/**
 * FaceDetector Component
 * Wrapper component for face detection with visual feedback
 */

import type { RefObject } from 'preact';
import { useFaceDetection } from '../../hooks/useFaceDetection';
import { STATE_COLORS, STATE_MESSAGES } from '../../config/detection';
import type { DetectionState } from '../../config/detection';
import styles from './FaceDetector.module.css';

export interface FaceDetectorProps {
  /** Reference to video element */
  videoRef: RefObject<HTMLVideoElement>;

  /** Enable auto start */
  autoStart?: boolean;

  /** Callback when face becomes ready for capture */
  onFaceReady?: () => void;

  /** Callback when countdown completes */
  onCapture?: () => void;

  /** Callback when detection state changes */
  onStateChange?: (state: DetectionState) => void;

  /** Callback when models fail to load (show manual capture) */
  onLoadError?: (error: Error) => void;

  /** Callback for manual capture button click */
  onManualCapture?: () => void;

  /** Custom class name */
  className?: string;
}

/**
 * FaceDetector - Visual face detection overlay
 */
export function FaceDetector({
  videoRef,
  autoStart = true,
  onFaceReady,
  onCapture,
  onStateChange,
  onLoadError,
  onManualCapture,
  className,
}: FaceDetectorProps) {
  const {
    state: detectionState,
  } = useFaceDetection({
    videoRef,
    autoStart,
    onFaceReady,
    onCountdownComplete: onCapture,
    onStateChange,
    onError: onLoadError,
  });

  const { state, countdown, loadingProgress, hasError } = detectionState;

  // Get visual state
  const borderColor = STATE_COLORS[state] || STATE_COLORS.initializing;
  const message = state === 'countdown' && countdown !== null
    ? `${countdown}...`
    : STATE_MESSAGES[state] || '';

  // Determine if we should show pulsing animation
  const isPulsing = state === 'countdown' || state === 'ready';

  // Show manual capture button when there's an error loading models
  const showManualCapture = hasError && onManualCapture;

  // Get CSS class based on state
  const getStateClass = () => {
    switch (state) {
      case 'ready':
      case 'countdown':
        return styles.ready;
      case 'face_too_small':
      case 'face_too_large':
      case 'face_not_centered':
      case 'poor_lighting':
        return styles.positioning;
      case 'multiple_faces':
      case 'error':
        return styles.error;
      default:
        return styles.idle;
    }
  };

  return (
    <div className={`${styles.container} ${className || ''}`}>
      {/* Face guide oval */}
      <div
        className={`${styles.faceGuide} ${getStateClass()} ${isPulsing ? styles.pulsing : ''}`}
        style={{ borderColor }}
      >
        {/* Countdown number */}
        {state === 'countdown' && countdown !== null && (
          <span className={styles.countdownNumber}>{countdown}</span>
        )}

        {/* Loading indicator */}
        {state === 'initializing' && !hasError && (
          <div className={styles.loadingIndicator}>
            <div
              className={styles.loadingBar}
              style={{ width: `${loadingProgress}%` }}
            />
          </div>
        )}
      </div>

      {/* Status message */}
      <p className={styles.message} style={{ color: borderColor }}>
        {message}
      </p>

      {/* Manual capture button (fallback when face detection fails to load) */}
      {showManualCapture && (
        <button
          type="button"
          className={styles.manualCaptureButton}
          onClick={onManualCapture}
        >
          <CameraIcon />
          <span>Capturar foto</span>
        </button>
      )}
    </div>
  );
}

/**
 * Camera icon for manual capture button
 */
function CameraIcon() {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z" />
      <circle cx="12" cy="13" r="4" />
    </svg>
  );
}

export default FaceDetector;
