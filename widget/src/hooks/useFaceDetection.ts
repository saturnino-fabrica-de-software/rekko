/**
 * useFaceDetection Hook
 * Manages face detection lifecycle with face-api.js
 */

import { useState, useEffect, useCallback, useRef } from 'preact/hooks';
import type { RefObject } from 'preact';
import {
  loadModels,
  detectFaces,
  areModelsLoaded,
} from '../services/faceDetection';
import {
  DEFAULT_DETECTION_CONFIG,
  getOptimalFps,
  getDetectionIntervalMs,
} from '../config/detection';
import type { DetectionState, DetectionConfig } from '../config/detection';
import type {
  FaceDetectionResult,
  FaceDetectionState,
  UseFaceDetectionReturn,
} from '../types/faceDetection';

export interface UseFaceDetectionOptions {
  videoRef: RefObject<HTMLVideoElement>;
  autoStart?: boolean;
  configOverrides?: Partial<DetectionConfig>;
  onFaceReady?: () => void;
  onStateChange?: (state: DetectionState) => void;
  onCountdownComplete?: () => void;
  onError?: (error: Error) => void;
}

/**
 * Hook for managing face detection with face-api.js
 */
export function useFaceDetection(options: UseFaceDetectionOptions): UseFaceDetectionReturn {
  const {
    videoRef,
    autoStart = true,
    configOverrides,
    onFaceReady,
    onStateChange,
    onCountdownComplete,
    onError,
  } = options;

  // Merge config with overrides
  const config = { ...DEFAULT_DETECTION_CONFIG, ...configOverrides };

  // State
  const [state, setState] = useState<FaceDetectionState>({
    state: 'initializing',
    detection: null,
    isReady: false,
    isDetecting: false,
    hasError: false,
    errorMessage: null,
    loadingProgress: 0,
    stabilityTime: 0,
    countdown: null,
  });

  // Refs for cleanup and interval management
  const detectionIntervalRef = useRef<number | null>(null);
  const countdownIntervalRef = useRef<number | null>(null);
  const stabilityStartRef = useRef<number | null>(null);
  const lastDetectionRef = useRef<FaceDetectionResult | null>(null);
  const isRunningRef = useRef(false);

  // Determine detection state from result
  const getStateFromDetection = useCallback((detection: FaceDetectionResult): DetectionState => {
    if (!detection.detected) {
      return 'no_face';
    }

    if (detection.faceCount > 1) {
      return 'multiple_faces';
    }

    const position = detection.position;
    const quality = detection.quality;

    if (!position || !quality) {
      return 'no_face';
    }

    // Check size
    if (position.sizeRatio < config.minFaceSize) {
      return 'face_too_small';
    }
    if (position.sizeRatio > config.maxFaceSize) {
      return 'face_too_large';
    }

    // Check centering
    if (!position.isCentered) {
      return 'face_not_centered';
    }

    // Check quality
    if (!quality.isAcceptable) {
      return 'poor_lighting';
    }

    return 'ready';
  }, [config.minFaceSize, config.maxFaceSize]);

  // Process detection result
  const processDetection = useCallback((detection: FaceDetectionResult) => {
    lastDetectionRef.current = detection;

    const newDetectionState = getStateFromDetection(detection);

    // Handle stability tracking
    if (newDetectionState === 'ready') {
      if (stabilityStartRef.current === null) {
        stabilityStartRef.current = Date.now();
      }

      const stabilityTime = Date.now() - stabilityStartRef.current;

      setState(prev => {
        const nextState = prev.state === 'countdown' ? 'countdown' : newDetectionState;
        if (prev.state !== nextState) {
          onStateChange?.(nextState);
        }
        return {
          ...prev,
          detection,
          state: nextState,
          stabilityTime,
        };
      });

      // Check if stable enough to start countdown
      if (stabilityTime >= config.stabilityTimeMs && state.state !== 'countdown') {
        onFaceReady?.();
        startCountdown();
      }
    } else {
      // Face not ready, reset stability
      stabilityStartRef.current = null;

      // Cancel countdown if face leaves
      if (state.countdown !== null) {
        cancelCountdown();
      }

      setState(prev => ({
        ...prev,
        detection,
        state: newDetectionState,
        stabilityTime: 0,
      }));
    }
  }, [getStateFromDetection, config.stabilityTimeMs, state.state, state.countdown, onFaceReady]);

  // Start countdown
  const startCountdown = useCallback(() => {
    if (countdownIntervalRef.current !== null) return;

    setState(prev => ({
      ...prev,
      state: 'countdown',
      countdown: config.countdownSeconds,
    }));

    countdownIntervalRef.current = window.setInterval(() => {
      setState(prev => {
        if (prev.countdown === null || prev.countdown <= 1) {
          // Countdown complete
          if (countdownIntervalRef.current) {
            clearInterval(countdownIntervalRef.current);
            countdownIntervalRef.current = null;
          }

          onCountdownComplete?.();

          return {
            ...prev,
            state: 'capturing',
            countdown: null,
          };
        }

        return {
          ...prev,
          countdown: prev.countdown - 1,
        };
      });
    }, 1000);
  }, [config.countdownSeconds, onCountdownComplete]);

  // Cancel countdown
  const cancelCountdown = useCallback(() => {
    if (countdownIntervalRef.current !== null) {
      clearInterval(countdownIntervalRef.current);
      countdownIntervalRef.current = null;
    }

    setState(prev => ({
      ...prev,
      countdown: null,
    }));
  }, []);

  // Run detection loop
  const runDetection = useCallback(async () => {
    const video = videoRef.current;
    if (!video || video.readyState !== 4 || !isRunningRef.current) {
      return;
    }

    try {
      const detection = await detectFaces(video);
      processDetection(detection);
    } catch (error) {
      console.error('Detection error:', error);
    }
  }, [videoRef, processDetection]);

  // Start detection
  const start = useCallback(async () => {
    if (isRunningRef.current) return;

    setState(prev => ({
      ...prev,
      state: 'initializing',
      hasError: false,
      errorMessage: null,
    }));

    try {
      // Load models if not already loaded
      if (!areModelsLoaded()) {
        await loadModels(undefined, (progress) => {
          setState(prev => ({
            ...prev,
            loadingProgress: progress,
          }));
        });
      }

      setState(prev => ({
        ...prev,
        isReady: true,
        state: 'no_face',
      }));

      // Start detection loop
      isRunningRef.current = true;
      setState(prev => ({ ...prev, isDetecting: true }));

      const fps = getOptimalFps(config);
      const intervalMs = getDetectionIntervalMs(fps);

      detectionIntervalRef.current = window.setInterval(runDetection, intervalMs);

      // Run first detection immediately
      await runDetection();
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to load models';

      setState(prev => ({
        ...prev,
        state: 'error',
        hasError: true,
        errorMessage,
        isReady: false,
      }));

      onError?.(error instanceof Error ? error : new Error(errorMessage));
    }
  }, [config, runDetection, onError]);

  // Stop detection
  const stop = useCallback(() => {
    isRunningRef.current = false;

    if (detectionIntervalRef.current !== null) {
      clearInterval(detectionIntervalRef.current);
      detectionIntervalRef.current = null;
    }

    cancelCountdown();

    setState(prev => ({
      ...prev,
      isDetecting: false,
    }));
  }, [cancelCountdown]);

  // Reset state
  const reset = useCallback(() => {
    stop();
    stabilityStartRef.current = null;
    lastDetectionRef.current = null;

    setState({
      state: 'initializing',
      detection: null,
      isReady: false,
      isDetecting: false,
      hasError: false,
      errorMessage: null,
      loadingProgress: 0,
      stabilityTime: 0,
      countdown: null,
    });
  }, [stop]);

  // Get current detection
  const getDetection = useCallback(() => {
    return lastDetectionRef.current;
  }, []);

  // Auto-start on mount
  useEffect(() => {
    if (autoStart) {
      start();
    }

    return () => {
      stop();
    };
  }, [autoStart, start, stop]);

  return {
    state,
    start,
    stop,
    reset,
    startCountdown,
    cancelCountdown,
    getDetection,
  };
}

export default useFaceDetection;
