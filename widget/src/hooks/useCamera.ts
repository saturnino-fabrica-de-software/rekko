import { useState, useEffect, useRef, useCallback } from 'preact/hooks';
import type { RekkoError } from '@/types';
import { Errors } from '@/errors';

export interface UseCameraOptions {
  facingMode?: 'user' | 'environment';
  width?: number;
  height?: number;
}

export interface UseCameraResult {
  videoRef: preact.RefObject<HTMLVideoElement>;
  stream: MediaStream | null;
  isReady: boolean;
  error: RekkoError | null;
  startCamera: () => Promise<void>;
  stopCamera: () => void;
  captureFrame: () => string | null;
}

export function useCamera(options: UseCameraOptions = {}): UseCameraResult {
  const { facingMode = 'user', width = 640, height = 480 } = options;

  const videoRef = useRef<HTMLVideoElement>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const [isReady, setIsReady] = useState(false);
  const [error, setError] = useState<RekkoError | null>(null);

  const stopCamera = useCallback(() => {
    if (streamRef.current) {
      streamRef.current.getTracks().forEach((track) => track.stop());
      streamRef.current = null;
    }
    setIsReady(false);
  }, []);

  const startCamera = useCallback(async () => {
    try {
      setError(null);

      const stream = await navigator.mediaDevices.getUserMedia({
        video: {
          facingMode,
          width: { ideal: width },
          height: { ideal: height },
        },
        audio: false,
      });

      streamRef.current = stream;

      if (videoRef.current) {
        videoRef.current.srcObject = stream;
        await videoRef.current.play();
        setIsReady(true);
      }
    } catch (err) {
      const rekkoError = handleCameraError(err);
      setError(rekkoError);
    }
  }, [facingMode, width, height]);

  const captureFrame = useCallback((): string | null => {
    if (!videoRef.current || !isReady) return null;

    const video = videoRef.current;
    const canvas = document.createElement('canvas');
    canvas.width = video.videoWidth;
    canvas.height = video.videoHeight;

    const ctx = canvas.getContext('2d');
    if (!ctx) return null;

    ctx.drawImage(video, 0, 0);
    return canvas.toDataURL('image/jpeg', 0.9);
  }, [isReady]);

  useEffect(() => {
    return () => {
      stopCamera();
    };
  }, [stopCamera]);

  return {
    videoRef,
    stream: streamRef.current,
    isReady,
    error,
    startCamera,
    stopCamera,
    captureFrame,
  };
}

function handleCameraError(err: unknown): RekkoError {
  if (err instanceof DOMException) {
    switch (err.name) {
      case 'NotAllowedError':
      case 'PermissionDeniedError':
        return Errors.cameraDenied();
      case 'NotFoundError':
      case 'DevicesNotFoundError':
        return Errors.cameraNotFound();
      case 'NotReadableError':
      case 'TrackStartError':
        return Errors.cameraInUse();
      default:
        return Errors.cameraError(err.message);
    }
  }
  return Errors.cameraError(err);
}
