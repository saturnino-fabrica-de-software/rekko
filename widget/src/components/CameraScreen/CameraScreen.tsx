import { useEffect, useRef, useState, useCallback } from 'preact/hooks';
import type { LocaleTexts, RekkoError, RekkoEventType } from '@/types';
import { Errors } from '@/errors';
import { FaceDetector } from './FaceDetector';
import type { DetectionState } from '@/config/detection';
import styles from './CameraScreen.module.css';

interface CameraScreenProps {
  texts: LocaleTexts['camera'];
  errorTexts: LocaleTexts['errors'];
  onCapture: (imageData: string) => void;
  onError: (error: RekkoError) => void;
  onEvent: (type: RekkoEventType, data?: unknown) => void;
  autoCapture?: boolean;
}

type CameraState = 'initializing' | 'ready' | 'no_face' | 'face_detected' | 'countdown' | 'capturing' | 'error';

const COUNTDOWN_SECONDS = 5;
const FACE_DETECTION_INTERVAL = 200; // Check for face every 200ms

// FaceDetector API types (Chrome/Edge)
interface FaceDetectorFace {
  boundingBox: DOMRectReadOnly;
}

interface FaceDetectorAPI {
  detect(image: ImageBitmapSource): Promise<FaceDetectorFace[]>;
}

declare global {
  interface Window {
    FaceDetector?: new () => FaceDetectorAPI;
  }
}

export function CameraScreen({ texts, onCapture, onError, onEvent, autoCapture = true }: CameraScreenProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const countdownRef = useRef<number | null>(null);
  const detectionRef = useRef<number | null>(null);
  const faceDetectorRef = useRef<FaceDetectorAPI | null>(null);
  const countdownValueRef = useRef<number>(COUNTDOWN_SECONDS);

  const [cameraState, setCameraState] = useState<CameraState>('initializing');
  const [instruction, setInstruction] = useState(texts.instruction);
  const [countdown, setCountdown] = useState<number | null>(null);
  const [faceDetected, setFaceDetected] = useState(false);
  const [hasFaceDetector, setHasFaceDetector] = useState(false);
  const [useFaceApiDetector, setUseFaceApiDetector] = useState(true);

  const captureFrame = useCallback(() => {
    if (!videoRef.current || !canvasRef.current) return;

    const video = videoRef.current;
    const canvas = canvasRef.current;
    const ctx = canvas.getContext('2d');

    if (!ctx) return;

    canvas.width = video.videoWidth;
    canvas.height = video.videoHeight;
    ctx.drawImage(video, 0, 0);

    const imageData = canvas.toDataURL('image/jpeg', 0.9);
    setCameraState('capturing');
    setInstruction(texts.capturing);

    stopDetection();
    stopCamera();
    onCapture(imageData);
  }, [texts.capturing, onCapture]);

  const startCountdown = useCallback(() => {
    if (countdownRef.current) return;

    setCameraState('countdown');
    countdownValueRef.current = COUNTDOWN_SECONDS;
    setCountdown(COUNTDOWN_SECONDS);

    countdownRef.current = window.setInterval(() => {
      countdownValueRef.current -= 1;
      setCountdown(countdownValueRef.current);

      if (countdownValueRef.current <= 0) {
        stopCountdown();
        captureFrame();
      }
    }, 1000);
  }, [captureFrame]);

  const stopCountdown = useCallback(() => {
    if (countdownRef.current) {
      clearInterval(countdownRef.current);
      countdownRef.current = null;
    }
  }, []);

  const resetCountdown = useCallback(() => {
    stopCountdown();
    countdownValueRef.current = COUNTDOWN_SECONDS;
    setCountdown(null);
    setCameraState('no_face');
    setInstruction(texts.instruction);
  }, [stopCountdown, texts.instruction]);

  // Initialize face detector
  useEffect(() => {
    if (window.FaceDetector) {
      try {
        faceDetectorRef.current = new window.FaceDetector();
        setHasFaceDetector(true);
      } catch {
        // FaceDetector not supported
        faceDetectorRef.current = null;
        setHasFaceDetector(false);
      }
    } else {
      setHasFaceDetector(false);
    }
  }, []);

  // Face detection loop
  const detectFace = useCallback(async () => {
    if (!videoRef.current || cameraState === 'capturing' || cameraState === 'error') {
      return;
    }

    const video = videoRef.current;
    if (video.readyState !== 4) return; // Video not ready

    let hasFace = false;

    // Try native FaceDetector API first
    if (faceDetectorRef.current) {
      try {
        const faces = await faceDetectorRef.current.detect(video);
        hasFace = faces.length > 0;
      } catch {
        // Detection failed, assume no face
        hasFace = false;
      }
    } else {
      // No FaceDetector API available - cannot auto-detect face
      // User must use manual capture button
      hasFace = false;
    }

    setFaceDetected(hasFace);

    if (hasFace) {
      if (cameraState === 'ready' || cameraState === 'no_face') {
        // Face appeared - start countdown
        onEvent('face_detected');
        setCameraState('countdown');
        setInstruction(texts.positioning);
        startCountdown();
      }
    } else {
      if (cameraState === 'countdown') {
        // Face disappeared - reset countdown to 5 seconds
        onEvent('face_lost');
        resetCountdown();
      }
    }
  }, [cameraState, onEvent, texts.positioning, texts.instruction, startCountdown, resetCountdown]);

  const startDetection = useCallback(() => {
    if (detectionRef.current) return;

    detectionRef.current = window.setInterval(() => {
      detectFace();
    }, FACE_DETECTION_INTERVAL);
  }, [detectFace]);

  const stopDetection = useCallback(() => {
    if (detectionRef.current) {
      clearInterval(detectionRef.current);
      detectionRef.current = null;
    }
  }, []);

  useEffect(() => {
    startCamera();

    return () => {
      stopCamera();
      stopCountdown();
      stopDetection();
    };
  }, []);

  // Start face detection when camera is ready (only if native FaceDetector is available and face-api.js is not being used)
  useEffect(() => {
    // Only use native detection when face-api.js is not active
    if (autoCapture && hasFaceDetector && !useFaceApiDetector && cameraState === 'ready') {
      setCameraState('no_face');
      startDetection();
    }
  }, [autoCapture, hasFaceDetector, useFaceApiDetector, cameraState, startDetection]);

  const startCamera = async () => {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        video: {
          facingMode: 'user',
          width: { ideal: 640 },
          height: { ideal: 480 },
        },
        audio: false,
      });

      streamRef.current = stream;

      if (videoRef.current) {
        videoRef.current.srcObject = stream;
        await videoRef.current.play();
        setCameraState('ready');
        setInstruction(texts.instruction);
        onEvent('camera_ready');
      }
    } catch (err) {
      handleCameraError(err);
    }
  };

  const stopCamera = () => {
    if (streamRef.current) {
      streamRef.current.getTracks().forEach((track) => track.stop());
      streamRef.current = null;
    }
  };

  const handleCameraError = (err: unknown) => {
    setCameraState('error');

    if (err instanceof DOMException) {
      switch (err.name) {
        case 'NotAllowedError':
        case 'PermissionDeniedError':
          onError(Errors.cameraDenied());
          break;
        case 'NotFoundError':
        case 'DevicesNotFoundError':
          onError(Errors.cameraNotFound());
          break;
        case 'NotReadableError':
        case 'TrackStartError':
          onError(Errors.cameraInUse());
          break;
        default:
          onError(Errors.cameraError(err.message));
      }
    } else {
      onError(Errors.cameraError(err));
    }

    onEvent('camera_error', err);
  };

  const handleManualCapture = () => {
    if (cameraState === 'ready' || cameraState === 'no_face' || cameraState === 'face_detected' || cameraState === 'countdown') {
      captureFrame();
    }
  };

  // Face-api.js detector callbacks
  const handleFaceApiStateChange = useCallback((state: DetectionState) => {
    if (state === 'ready' || state === 'countdown') {
      setFaceDetected(true);
    } else {
      setFaceDetected(false);
    }
    onEvent('face_detection_state', state);
  }, [onEvent]);

  const handleFaceApiReady = useCallback(() => {
    onEvent('face_detected');
  }, [onEvent]);

  const handleFaceApiCapture = useCallback(() => {
    captureFrame();
  }, [captureFrame]);

  const handleFaceApiLoadError = useCallback(() => {
    setUseFaceApiDetector(false);
  }, []);

  const getGuideClass = () => {
    const classes = [styles.faceGuide];
    if (cameraState === 'countdown' && faceDetected) classes.push(styles.countdown);
    if (cameraState === 'no_face') classes.push(styles.noFace);
    if (faceDetected && cameraState !== 'no_face') classes.push(styles.detected);
    return classes.join(' ');
  };

  const getInstructionText = () => {
    if (cameraState === 'countdown' && countdown !== null && countdown > 0) {
      return texts.capturing;
    }
    if (cameraState === 'no_face') {
      return texts.instruction;
    }
    return instruction;
  };

  return (
    <div class={styles.container}>
      <h2 class={styles.title}>{texts.title}</h2>

      <div class={styles.cameraContainer}>
        <video
          ref={videoRef}
          class={styles.video}
          playsInline
          muted
          autoPlay
        />
        <canvas ref={canvasRef} class={styles.canvas} />

        <div class={styles.overlay}>
          {/* Use face-api.js detector when available and autoCapture is enabled */}
          {autoCapture && useFaceApiDetector && cameraState !== 'initializing' && cameraState !== 'error' ? (
            <FaceDetector
              videoRef={videoRef}
              autoStart={cameraState === 'ready'}
              onFaceReady={handleFaceApiReady}
              onCapture={handleFaceApiCapture}
              onStateChange={handleFaceApiStateChange}
              onLoadError={handleFaceApiLoadError}
              onManualCapture={handleManualCapture}
            />
          ) : (
            <div class={getGuideClass()}>
              {countdown !== null && countdown > 0 && faceDetected && (
                <span class={styles.countdownNumber}>{countdown}</span>
              )}
            </div>
          )}
        </div>

        {cameraState === 'initializing' && (
          <div class={styles.loadingOverlay}>
            <Spinner />
          </div>
        )}
      </div>

      <p class={styles.instruction}>
        {getInstructionText()}
      </p>

      {/* Show manual capture button when not using auto detection */}
      {(!autoCapture || (!useFaceApiDetector && !hasFaceDetector)) && (cameraState === 'ready' || cameraState === 'no_face' || cameraState === 'face_detected') && (
        <button class={styles.captureButton} onClick={handleManualCapture}>
          <CameraIcon />
        </button>
      )}
    </div>
  );
}

function Spinner() {
  return (
    <svg class={styles.spinner} viewBox="0 0 24 24" fill="none">
      <circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="2" opacity="0.25" />
      <path
        d="M12 2a10 10 0 0 1 10 10"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
      />
    </svg>
  );
}

function CameraIcon() {
  return (
    <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <circle cx="12" cy="12" r="10" />
    </svg>
  );
}
