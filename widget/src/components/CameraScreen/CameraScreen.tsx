import { useEffect, useRef, useState } from 'preact/hooks';
import type { LocaleTexts, RekkoError, RekkoEventType } from '@/types';
import { Errors } from '@/errors';
import styles from './CameraScreen.module.css';

interface CameraScreenProps {
  texts: LocaleTexts['camera'];
  errorTexts: LocaleTexts['errors'];
  onCapture: (imageData: string) => void;
  onError: (error: RekkoError) => void;
  onEvent: (type: RekkoEventType, data?: unknown) => void;
}

type CameraState = 'initializing' | 'ready' | 'detecting' | 'capturing' | 'error';

export function CameraScreen({ texts, onCapture, onError, onEvent }: CameraScreenProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const [cameraState, setCameraState] = useState<CameraState>('initializing');
  const [instruction, setInstruction] = useState(texts.instruction);

  useEffect(() => {
    startCamera();

    return () => {
      stopCamera();
    };
  }, []);

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

  const captureFrame = () => {
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

    stopCamera();
    onCapture(imageData);
  };

  const handleManualCapture = () => {
    if (cameraState === 'ready' || cameraState === 'detecting') {
      captureFrame();
    }
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
          <div class={`${styles.faceGuide} ${cameraState === 'detecting' ? styles.detected : ''}`} />
        </div>

        {cameraState === 'initializing' && (
          <div class={styles.loadingOverlay}>
            <Spinner />
          </div>
        )}
      </div>

      <p class={styles.instruction}>{instruction}</p>

      {(cameraState === 'ready' || cameraState === 'detecting') && (
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
