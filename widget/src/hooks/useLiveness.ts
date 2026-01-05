/**
 * useLiveness Hook
 * Manages active liveness detection challenges (turn head, blink, etc.)
 */

import { useState, useCallback, useRef, useEffect } from 'preact/hooks';
import type { RefObject } from 'preact';
import { detectFaces, areModelsLoaded } from '../services/faceDetection';

/**
 * Available liveness challenges
 */
export type LivenessChallenge = 'turn_left' | 'turn_right' | 'blink';

/**
 * Challenge state
 */
export type ChallengeState = 'pending' | 'active' | 'success' | 'failed' | 'timeout';

/**
 * Challenge result
 */
export interface ChallengeResult {
  challenge: LivenessChallenge;
  success: boolean;
  duration: number;
  frames: string[];
}

/**
 * Liveness state
 */
export interface LivenessState {
  isActive: boolean;
  currentChallenge: LivenessChallenge | null;
  challengeState: ChallengeState;
  completedChallenges: ChallengeResult[];
  timeRemaining: number;
  attempt: number;
  progress: number;
  waitingForNeutral: boolean;
}

/**
 * Liveness config
 */
export interface LivenessConfig {
  challenges: LivenessChallenge[];
  timeoutMs: number;
  maxAttempts: number;
  turnThreshold: number;
  blinkFrames: number;
}

const DEFAULT_CONFIG: LivenessConfig = {
  challenges: ['turn_right', 'turn_left'],
  timeoutMs: 10000,
  maxAttempts: 3,
  turnThreshold: 15,
  blinkFrames: 3,
};

const NEUTRAL_YAW_THRESHOLD = 8;

export interface UseLivenessOptions {
  videoRef: RefObject<HTMLVideoElement>;
  config?: Partial<LivenessConfig>;
  onChallengeStart?: (challenge: LivenessChallenge) => void;
  onChallengeComplete?: (result: ChallengeResult) => void;
  onComplete?: (results: ChallengeResult[]) => void;
  onFailed?: () => void;
}

export interface UseLivenessReturn {
  state: LivenessState;
  start: () => void;
  stop: () => void;
  reset: () => void;
  getFrames: () => string[];
}

/**
 * Hook for active liveness detection
 */
export function useLiveness(options: UseLivenessOptions): UseLivenessReturn {
  const {
    videoRef,
    config: configOverrides,
    onChallengeStart,
    onChallengeComplete,
    onComplete,
    onFailed,
  } = options;

  const config = { ...DEFAULT_CONFIG, ...configOverrides };

  const [state, setState] = useState<LivenessState>({
    isActive: false,
    currentChallenge: null,
    challengeState: 'pending',
    completedChallenges: [],
    timeRemaining: config.timeoutMs,
    attempt: 1,
    progress: 0,
    waitingForNeutral: false,
  });

  const detectionRef = useRef<number | null>(null);
  const timerRef = useRef<number | null>(null);
  const framesRef = useRef<string[]>([]);
  const startTimeRef = useRef<number>(0);
  const baseYawRef = useRef<number | null>(null);
  const blinkCountRef = useRef<number>(0);
  const eyeOpenRef = useRef<boolean>(true);
  const waitingForNeutralRef = useRef<boolean>(false);

  // Ref para quebrar dependência circular entre completeChallenge e runChallenge
  const runChallengeRef = useRef<(challenge: LivenessChallenge, completed?: ChallengeResult[]) => void>(() => {});

  const stopTimers = useCallback(() => {
    if (detectionRef.current) {
      clearInterval(detectionRef.current);
      detectionRef.current = null;
    }
    if (timerRef.current) {
      clearInterval(timerRef.current);
      timerRef.current = null;
    }
  }, []);

  const captureFrame = useCallback(() => {
    const video = videoRef.current;
    if (!video) return;

    const canvas = document.createElement('canvas');
    canvas.width = video.videoWidth;
    canvas.height = video.videoHeight;
    const ctx = canvas.getContext('2d');
    if (ctx) {
      ctx.drawImage(video, 0, 0);
      framesRef.current.push(canvas.toDataURL('image/jpeg', 0.8));
    }
  }, [videoRef]);

  const calculateYaw = useCallback((landmarks: {
    leftEye: { x: number; y: number };
    rightEye: { x: number; y: number };
    nose: { x: number; y: number };
  }) => {
    const eyeCenter = (landmarks.leftEye.x + landmarks.rightEye.x) / 2;
    const eyeDistance = Math.abs(landmarks.rightEye.x - landmarks.leftEye.x);
    const noseOffset = landmarks.nose.x - eyeCenter;
    return (noseOffset / eyeDistance) * 60;
  }, []);

  const completeChallenge = useCallback((success: boolean) => {
    stopTimers();

    const result: ChallengeResult = {
      challenge: state.currentChallenge!,
      success,
      duration: Date.now() - startTimeRef.current,
      frames: [...framesRef.current],
    };

    const completed = [...state.completedChallenges, result];
    const remaining = config.challenges.slice(completed.length);

    onChallengeComplete?.(result);

    if (!success) {
      if (state.attempt < config.maxAttempts) {
        setState(prev => ({
          ...prev,
          challengeState: 'failed',
          attempt: prev.attempt + 1,
        }));
      } else {
        setState(prev => ({
          ...prev,
          isActive: false,
          challengeState: 'failed',
          completedChallenges: completed,
        }));
        onFailed?.();
      }
      return;
    }

    if (remaining.length === 0) {
      setState(prev => ({
        ...prev,
        isActive: false,
        currentChallenge: null,
        challengeState: 'success',
        completedChallenges: completed,
        progress: 100,
      }));
      onComplete?.(completed);
    } else {
      const nextChallenge = remaining[0];
      if (nextChallenge) {
        // Usar ref para evitar dependência circular
        runChallengeRef.current(nextChallenge, completed);
      }
    }
  }, [state, config, onChallengeComplete, onComplete, onFailed, stopTimers]);

  const runChallenge = useCallback((challenge: LivenessChallenge, completed: ChallengeResult[] = []) => {
    stopTimers();
    framesRef.current = [];
    baseYawRef.current = null;
    blinkCountRef.current = 0;
    eyeOpenRef.current = true;
    startTimeRef.current = Date.now();

    // If this is not the first challenge, wait for user to return to neutral position
    const shouldWaitForNeutral = completed.length > 0;
    waitingForNeutralRef.current = shouldWaitForNeutral;

    setState(prev => ({
      ...prev,
      isActive: true,
      currentChallenge: challenge,
      challengeState: 'active',
      completedChallenges: completed,
      timeRemaining: config.timeoutMs,
      progress: (completed.length / config.challenges.length) * 100,
      waitingForNeutral: shouldWaitForNeutral,
    }));

    onChallengeStart?.(challenge);

    timerRef.current = window.setInterval(() => {
      const elapsed = Date.now() - startTimeRef.current;
      const remaining = Math.max(0, config.timeoutMs - elapsed);

      setState(prev => ({ ...prev, timeRemaining: remaining }));

      if (remaining === 0) {
        completeChallenge(false);
      }
    }, 100);

    const detect = async () => {
      if (!videoRef.current || !areModelsLoaded()) return;

      try {
        const detection = await detectFaces(videoRef.current);
        if (!detection.detected || !detection.landmarks) return;

        captureFrame();

        if (challenge === 'turn_left' || challenge === 'turn_right') {
          const yaw = calculateYaw(detection.landmarks);

          // If waiting for neutral position (between challenges), check if user returned to center
          if (waitingForNeutralRef.current) {
            if (Math.abs(yaw) < NEUTRAL_YAW_THRESHOLD) {
              waitingForNeutralRef.current = false;
              baseYawRef.current = yaw;
              setState(prev => ({ ...prev, waitingForNeutral: false }));
            }
            return;
          }

          if (baseYawRef.current === null) {
            baseYawRef.current = yaw;
            return;
          }

          const diff = yaw - baseYawRef.current;
          const threshold = config.turnThreshold;

          if (challenge === 'turn_right' && diff < -threshold) {
            completeChallenge(true);
          } else if (challenge === 'turn_left' && diff > threshold) {
            completeChallenge(true);
          }
        }

        if (challenge === 'blink') {
          const eyeAspect = detection.quality?.eyesVisible ?? true;

          if (eyeOpenRef.current && !eyeAspect) {
            blinkCountRef.current++;
          }
          eyeOpenRef.current = eyeAspect;

          if (blinkCountRef.current >= config.blinkFrames) {
            completeChallenge(true);
          }
        }
      } catch {
        // Detection error, continue
      }
    };

    detectionRef.current = window.setInterval(detect, 150);
  }, [config, videoRef, onChallengeStart, captureFrame, calculateYaw, completeChallenge, stopTimers]);

  // Atualizar ref sempre que runChallenge mudar
  runChallengeRef.current = runChallenge;

  const start = useCallback(() => {
    if (state.isActive || config.challenges.length === 0) return;
    const firstChallenge = config.challenges[0];
    if (!firstChallenge) return;
    framesRef.current = [];
    runChallenge(firstChallenge);
  }, [state.isActive, config.challenges, runChallenge]);

  const stop = useCallback(() => {
    stopTimers();
    setState(prev => ({
      ...prev,
      isActive: false,
      currentChallenge: null,
      challengeState: 'pending',
    }));
  }, [stopTimers]);

  const reset = useCallback(() => {
    stopTimers();
    framesRef.current = [];
    baseYawRef.current = null;
    blinkCountRef.current = 0;
    waitingForNeutralRef.current = false;

    setState({
      isActive: false,
      currentChallenge: null,
      challengeState: 'pending',
      completedChallenges: [],
      timeRemaining: config.timeoutMs,
      attempt: 1,
      progress: 0,
      waitingForNeutral: false,
    });
  }, [config.timeoutMs, stopTimers]);

  const getFrames = useCallback(() => [...framesRef.current], []);

  useEffect(() => {
    return () => stopTimers();
  }, [stopTimers]);

  return {
    state,
    start,
    stop,
    reset,
    getFrames,
  };
}

export default useLiveness;
