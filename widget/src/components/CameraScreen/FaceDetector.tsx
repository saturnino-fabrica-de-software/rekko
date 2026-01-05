/**
 * FaceDetector Component
 * Clean overlay with oval mask and progress border
 */

import type { RefObject } from 'preact';
import { useFaceDetection } from '../../hooks/useFaceDetection';
import { STATE_COLORS, STATE_MESSAGES, DEFAULT_DETECTION_CONFIG } from '../../config/detection';
import type { DetectionState } from '../../config/detection';
import styles from './FaceDetector.module.css';

export interface FaceDetectorProps {
  videoRef: RefObject<HTMLVideoElement>;
  autoStart?: boolean;
  onFaceReady?: () => void;
  onCapture?: () => void;
  onStateChange?: (state: DetectionState) => void;
  onLoadError?: (error: Error) => void;
  onManualCapture?: () => void;
  className?: string;
}

// Oval dimensions
const OVAL_WIDTH = 200;
const OVAL_HEIGHT = 260;
const STROKE_WIDTH = 5;

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
  const { state: detectionState } = useFaceDetection({
    videoRef,
    autoStart,
    onFaceReady,
    onCountdownComplete: onCapture,
    onStateChange,
    onError: onLoadError,
  });

  const { state, loadingProgress, hasError, countdown } = detectionState;
  const countdownSeconds = DEFAULT_DETECTION_CONFIG.countdownSeconds;
  const showManualCapture = hasError && onManualCapture;

  // Calculate ellipse perimeter for stroke animation
  const rx = (OVAL_WIDTH - STROKE_WIDTH) / 2;
  const ry = (OVAL_HEIGHT - STROKE_WIDTH) / 2;
  const h = Math.pow((rx - ry) / (rx + ry), 2);
  const perimeter = Math.PI * (rx + ry) * (1 + (3 * h) / (10 + Math.sqrt(4 - 3 * h)));

  // Determine colors and states
  const isCountdown = state === 'countdown';
  const isReady = state === 'ready';
  const isCapturing = state === 'capturing';
  const borderColor = STATE_COLORS[state] || STATE_COLORS.initializing;
  const message = STATE_MESSAGES[state] || '';

  // Calculate progress percentage (0 to 1)
  let progress = 0;
  if (isCapturing) {
    progress = 1; // 100% when capturing
  } else if (isCountdown && countdown !== null) {
    // Progress fills as countdown decreases: 3→0%, 2→33%, 1→66%
    progress = (countdownSeconds - countdown) / countdownSeconds;
  }
  const progressOffset = perimeter * (1 - progress);

  return (
    <div className={`${styles.container} ${className || ''}`}>
      {/* Dark overlay with oval cutout */}
      <svg className={styles.maskOverlay} viewBox="0 0 280 280" preserveAspectRatio="xMidYMid slice">
        <defs>
          <mask id="ovalMask">
            <rect width="100%" height="100%" fill="white" />
            <ellipse cx="140" cy="140" rx="100" ry="130" fill="black" />
          </mask>
        </defs>
        <rect width="100%" height="100%" fill="rgba(0, 0, 0, 0.6)" mask="url(#ovalMask)" />
      </svg>

      {/* Oval border with progress animation */}
      <svg
        className={styles.ovalBorder}
        width={OVAL_WIDTH}
        height={OVAL_HEIGHT}
        viewBox={`0 0 ${OVAL_WIDTH} ${OVAL_HEIGHT}`}
      >
        {/* Base border (dashed when idle, solid when active) */}
        <ellipse
          cx={OVAL_WIDTH / 2}
          cy={OVAL_HEIGHT / 2}
          rx={rx}
          ry={ry}
          fill="none"
          stroke={isCountdown ? 'rgba(255,255,255,0.2)' : borderColor}
          strokeWidth={STROKE_WIDTH}
          strokeDasharray={isReady || isCountdown || isCapturing ? 'none' : '8 8'}
        />

        {/* Progress border (fills during countdown) */}
        {(isReady || isCountdown || isCapturing) && (
          <ellipse
            className={styles.progressBorder}
            cx={OVAL_WIDTH / 2}
            cy={OVAL_HEIGHT / 2}
            rx={rx}
            ry={ry}
            fill="none"
            stroke="#10B981"
            strokeWidth={STROKE_WIDTH}
            strokeLinecap="round"
            strokeDasharray={perimeter}
            strokeDashoffset={progressOffset}
            transform={`rotate(-90 ${OVAL_WIDTH / 2} ${OVAL_HEIGHT / 2})`}
          />
        )}
      </svg>

      {/* Center content */}
      <div className={styles.centerContent}>
        {/* Loading bar */}
        {state === 'initializing' && !hasError && (
          <div className={styles.loadingIndicator}>
            <div className={styles.loadingBar} style={{ width: `${loadingProgress}%` }} />
          </div>
        )}

        {/* Checkmark on capture */}
        {isCapturing && (
          <div className={styles.captureIcon}>
            <svg width="56" height="56" viewBox="0 0 24 24" fill="none" stroke="#10B981" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
              <polyline points="20 6 9 17 4 12" />
            </svg>
          </div>
        )}
      </div>

      {/* Status message */}
      <p className={styles.message} style={{ color: borderColor }}>
        {message}
      </p>

      {/* Manual capture fallback */}
      {showManualCapture && (
        <button type="button" className={styles.manualCaptureButton} onClick={onManualCapture}>
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z" />
            <circle cx="12" cy="13" r="4" />
          </svg>
          <span>Capturar foto</span>
        </button>
      )}
    </div>
  );
}

export default FaceDetector;
