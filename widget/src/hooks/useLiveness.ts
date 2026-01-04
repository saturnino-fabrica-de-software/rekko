import { useState, useCallback } from 'preact/hooks';

export interface LivenessChecks {
  faceDetected: boolean;
  faceCentered: boolean;
  eyesOpen: boolean;
  goodLighting: boolean;
}

export interface UseLivenessResult {
  checks: LivenessChecks;
  isLive: boolean;
  confidence: number;
  analyze: (videoElement: HTMLVideoElement) => void;
  reset: () => void;
}

const initialChecks: LivenessChecks = {
  faceDetected: false,
  faceCentered: false,
  eyesOpen: false,
  goodLighting: false,
};

export function useLiveness(): UseLivenessResult {
  const [checks, setChecks] = useState<LivenessChecks>(initialChecks);

  const analyze = useCallback((_videoElement: HTMLVideoElement) => {
    // Simplified liveness detection
    // In production, this would use MediaPipe or similar
    // For now, we'll simulate positive checks after a short delay
    // Real implementation would analyze video frames

    setChecks({
      faceDetected: true,
      faceCentered: true,
      eyesOpen: true,
      goodLighting: true,
    });
  }, []);

  const reset = useCallback(() => {
    setChecks(initialChecks);
  }, []);

  const passedChecks = Object.values(checks).filter(Boolean).length;
  const totalChecks = Object.keys(checks).length;
  const confidence = passedChecks / totalChecks;
  const isLive = passedChecks === totalChecks;

  return {
    checks,
    isLive,
    confidence,
    analyze,
    reset,
  };
}
