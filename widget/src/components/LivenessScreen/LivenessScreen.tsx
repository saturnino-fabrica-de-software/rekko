/**
 * LivenessScreen Component
 * Active liveness detection with visual challenges
 */

import { useRef, useEffect, useState, useCallback } from 'preact/hooks';
import { useLiveness, type LivenessChallenge, type ChallengeResult } from '@/hooks/useLiveness';
import { loadModels, areModelsLoaded } from '@/services/faceDetection';
import styles from './LivenessScreen.module.css';

interface LivenessScreenProps {
  onComplete: (frames: string[]) => void;
  onFailed: () => void;
  onSkip?: () => void;
}

const CHALLENGE_TEXTS: Record<LivenessChallenge, { instruction: string; icon: string }> = {
  turn_left: {
    instruction: 'Vire a cabeça para a esquerda',
    icon: '←',
  },
  turn_right: {
    instruction: 'Vire a cabeça para a direita',
    icon: '→',
  },
  blink: {
    instruction: 'Pisque os olhos',
    icon: '👁',
  },
};

export function LivenessScreen({ onComplete, onFailed, onSkip }: LivenessScreenProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const streamRef = useRef<MediaStream | null>(null);

  const [isLoading, setIsLoading] = useState(true);
  const [cameraReady, setCameraReady] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleComplete = useCallback((results: ChallengeResult[]) => {
    const allFrames = results.flatMap(r => r.frames);
    onComplete(allFrames);
  }, [onComplete]);

  const { state, start, reset } = useLiveness({
    videoRef,
    onComplete: handleComplete,
    onFailed,
  });

  useEffect(() => {
    const init = async () => {
      try {
        setIsLoading(true);

        // Start camera
        const stream = await navigator.mediaDevices.getUserMedia({
          video: { facingMode: 'user', width: { ideal: 640 }, height: { ideal: 480 } },
          audio: false,
        });

        streamRef.current = stream;

        if (videoRef.current) {
          videoRef.current.srcObject = stream;
          await videoRef.current.play();
          setCameraReady(true);
        }

        // Load face detection models
        if (!areModelsLoaded()) {
          await loadModels();
        }

        setIsLoading(false);
      } catch (err) {
        setError('Não foi possível acessar a câmera');
        setIsLoading(false);
      }
    };

    init();

    return () => {
      if (streamRef.current) {
        streamRef.current.getTracks().forEach(t => t.stop());
      }
    };
  }, []);

  useEffect(() => {
    if (cameraReady && !isLoading && !state.isActive && state.challengeState === 'pending') {
      const timer = setTimeout(() => start(), 1000);
      return () => clearTimeout(timer);
    }
  }, [cameraReady, isLoading, state.isActive, state.challengeState, start]);

  const handleRetry = () => {
    reset();
    setTimeout(() => start(), 500);
  };

  const formatTime = (ms: number) => {
    const seconds = Math.ceil(ms / 1000);
    return `${seconds}s`;
  };

  const getCurrentInstruction = () => {
    if (!state.currentChallenge) return '';
    return CHALLENGE_TEXTS[state.currentChallenge].instruction;
  };

  const getCurrentIcon = () => {
    if (!state.currentChallenge) return '';
    return CHALLENGE_TEXTS[state.currentChallenge].icon;
  };

  if (error) {
    return (
      <div class={styles.container}>
        <div class={styles.errorState}>
          <div class={styles.errorIcon}>⚠️</div>
          <p class={styles.errorText}>{error}</p>
          {onSkip && (
            <button class={styles.skipButton} onClick={onSkip}>
              Pular verificação
            </button>
          )}
        </div>
      </div>
    );
  }

  return (
    <div class={styles.container}>
      <h2 class={styles.title}>Verificação de Identidade</h2>

      <div class={styles.videoContainer}>
        <video
          ref={videoRef}
          class={styles.video}
          playsInline
          muted
          autoPlay
        />

        <div class={styles.overlay}>
          <div class={`${styles.faceGuide} ${state.challengeState === 'success' ? styles.success : ''}`}>
            {state.isActive && (
              <span class={styles.challengeIcon}>{getCurrentIcon()}</span>
            )}
          </div>
        </div>

        {isLoading && (
          <div class={styles.loadingOverlay}>
            <div class={styles.spinner} />
            <p>Preparando...</p>
          </div>
        )}
      </div>

      <div class={styles.progressContainer}>
        <div class={styles.progressBar}>
          <div class={styles.progressFill} style={{ width: `${state.progress}%` }} />
        </div>
        <span class={styles.progressText}>
          {Math.round(state.progress)}%
        </span>
      </div>

      {state.isActive && (
        <div class={styles.instructionContainer}>
          <p class={styles.instruction}>{getCurrentInstruction()}</p>
          <div class={styles.timer}>
            <span class={styles.timerIcon}>⏱</span>
            <span>{formatTime(state.timeRemaining)}</span>
          </div>
        </div>
      )}

      {state.challengeState === 'failed' && (
        <div class={styles.failedState}>
          <p class={styles.failedText}>
            {state.attempt < 3
              ? 'Não detectado. Tente novamente.'
              : 'Verificação falhou. Por favor, tente novamente.'}
          </p>
          <button class={styles.retryButton} onClick={handleRetry}>
            Tentar Novamente
          </button>
        </div>
      )}

      {state.challengeState === 'success' && (
        <div class={styles.successState}>
          <div class={styles.successIcon}>✓</div>
          <p class={styles.successText}>Verificação concluída!</p>
        </div>
      )}

      {onSkip && !state.isActive && state.challengeState !== 'success' && (
        <button class={styles.skipLink} onClick={onSkip}>
          Pular esta etapa
        </button>
      )}
    </div>
  );
}

export default LivenessScreen;
