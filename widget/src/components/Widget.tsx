import { useEffect, useRef, useState } from 'preact/hooks';
import { Modal } from './Modal/Modal';
import { ConsentScreen } from './ConsentScreen/ConsentScreen';
import { OrientationScreen } from './OrientationScreen/OrientationScreen';
import { CameraScreen } from './CameraScreen/CameraScreen';
import { LivenessScreen } from './LivenessScreen/LivenessScreen';
import { ProcessingScreen } from './ProcessingScreen/ProcessingScreen';
import { ResultScreen } from './ResultScreen/ResultScreen';
import { ApiClient } from '@/services/api';
import { Errors, isRekkoError } from '@/errors';
import type { RekkoWidget } from '@/Rekko';
import type { RekkoResult, RekkoError } from '@/types';

interface WidgetProps {
  rekko: RekkoWidget;
}

/**
 * Maps server-side liveness failure reasons to user-friendly messages.
 * This ensures consistent UX regardless of backend provider (DeepFace/Rekognition).
 */
const LIVENESS_ERROR_MESSAGES: Record<string, string> = {
  'no face detected': 'Não foi possível detectar seu rosto. Certifique-se de estar bem iluminado e de frente para a câmera.',
  'multiple faces detected': 'Detectamos mais de um rosto. Por favor, certifique-se de que apenas você está na imagem.',
  'image quality too low': 'A qualidade da imagem está baixa. Tente melhorar a iluminação do ambiente.',
  'confidence below threshold': 'Não conseguimos validar com certeza. Tente novamente com melhor iluminação.',
  'eyes not visible': 'Seus olhos não estão visíveis. Remova óculos escuros e olhe para a câmera.',
  'face not centered': 'Seu rosto não está centralizado. Posicione-se no centro da tela.',
};

/**
 * Gets a user-friendly error message from server reasons
 */
function getLivenessErrorMessage(reasons: string[] | undefined, defaultMessage: string): string {
  if (!reasons || reasons.length === 0) {
    return defaultMessage;
  }

  // Try to find a mapped message for the first reason
  const firstReason = reasons[0];
  if (!firstReason) {
    return defaultMessage;
  }

  const reason = firstReason.toLowerCase();
  for (const [key, message] of Object.entries(LIVENESS_ERROR_MESSAGES)) {
    if (reason.includes(key)) {
      return message;
    }
  }

  // If no mapping found, return a generic but helpful message
  return 'A verificação não foi bem-sucedida. Tente novamente em um ambiente bem iluminado.';
}

export function Widget({ rekko }: WidgetProps) {
  const state = rekko.getState();
  const texts = rekko.getTexts();
  const config = rekko.getConfig();
  const options = rekko.getOptions();

  const apiClientRef = useRef<ApiClient | null>(null);
  const [result, setResult] = useState<RekkoResult | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [sessionReady, setSessionReady] = useState(false);
  const [capturedImage, setCapturedImage] = useState<string | null>(null);

  // Create API client and session on mount
  useEffect(() => {
    if (!config || !options) return;

    const initSession = async () => {
      try {
        const client = new ApiClient(config.apiUrl!, config.publicKey);
        apiClientRef.current = client;
        await client.createSession();
        setSessionReady(true);
      } catch (err) {
        rekko.handleError(err as Parameters<typeof rekko.handleError>[0]);
      }
    };

    initSession();
  }, [config, options]);

  if (!texts || !config || !options) return null;

  const handleClose = () => {
    rekko.close();
  };

  const processImage = async (imageData: string) => {
    rekko.emit('capture_started');
    rekko.setState('processing');

    const client = apiClientRef.current;
    if (!client) {
      rekko.handleError(Errors.sessionExpired());
      return;
    }

    try {
      rekko.emit('processing');

      const response = await client.process(
        options.mode,
        options.externalId!,
        imageData
      );

      if (options.mode === 'verify') {
        const verifyResult = response as { verified: boolean; confidence: number; faceId?: string };
        const rekkoResult: RekkoResult = {
          mode: 'verify',
          verified: verifyResult.verified,
          confidence: verifyResult.confidence,
          faceId: verifyResult.faceId,
          externalId: options.externalId,
        };
        setResult(rekkoResult);

        if (verifyResult.verified) {
          rekko.emit('verification_success', verifyResult);
        } else {
          rekko.emit('verification_failed', verifyResult);
        }
      } else if (options.mode === 'identify') {
        const searchResult = response as { identified: boolean; externalId?: string; faceId?: string; confidence?: number };
        const rekkoResult: RekkoResult = {
          mode: 'identify',
          identified: searchResult.identified,
          externalId: searchResult.externalId,
          faceId: searchResult.faceId,
          confidence: searchResult.confidence,
        };
        setResult(rekkoResult);

        if (searchResult.identified) {
          rekko.emit('identification_success', searchResult);
        } else {
          rekko.emit('identification_failed', searchResult);
        }
      } else {
        const registerResult = response as { faceId: string; registered: boolean };
        const rekkoResult: RekkoResult = {
          mode: 'register',
          registered: registerResult.registered,
          faceId: registerResult.faceId,
          externalId: options.externalId,
        };
        setResult(rekkoResult);

        if (registerResult.registered) {
          rekko.emit('registration_success', registerResult);
        } else {
          rekko.emit('registration_failed', registerResult);
        }
      }

      rekko.setState('result');
      setErrorMessage(null);
    } catch (err) {
      const error = err as RekkoError;
      setResult({ mode: options.mode, verified: false, registered: false, identified: false });

      if (isRekkoError(err)) {
        setErrorMessage(error.message);
      } else {
        setErrorMessage(null);
      }

      const failedEvent = options.mode === 'verify'
        ? 'verification_failed'
        : options.mode === 'identify'
          ? 'identification_failed'
          : 'registration_failed';
      rekko.emit(failedEvent, error);
      rekko.setState('result');
    }
  };

  const handleCameraCapture = (imageData: string) => {
    setCapturedImage(imageData);

    // Register mode requires liveness check first
    // Verify and Identify modes go straight to processing
    if (options.mode === 'register') {
      rekko.emit('liveness_started');
      rekko.setState('liveness');
    } else {
      processImage(imageData);
    }
  };

  const handleLivenessComplete = async (frames: string[]) => {
    const client = apiClientRef.current;
    const imageToProcess = capturedImage || (frames.length > 0 ? frames[0] : null);

    if (!imageToProcess) {
      handleLivenessFailed();
      return;
    }

    // Use the initial captured image for liveness validation - it was taken when
    // the face was centered and facing the camera. The liveness frames are taken
    // during head turns and may not have a detectable frontal face.
    const livenessImage = capturedImage || imageToProcess;

    if (client) {
      try {
        rekko.emit('liveness_challenge', { stage: 'server_validation' });
        const livenessResult = await client.validateLiveness(livenessImage);

        if (!livenessResult.isLive) {
          // Get user-friendly error message based on server reasons
          const userMessage = getLivenessErrorMessage(
            livenessResult.reasons,
            texts.liveness.failed
          );

          rekko.emit('liveness_failed', {
            reason: 'server_validation',
            confidence: livenessResult.confidence,
            serverReasons: livenessResult.reasons,
          });
          setResult({ mode: options.mode, verified: false, registered: false });
          setErrorMessage(userMessage);
          rekko.setState('result');
          return;
        }

        rekko.emit('liveness_success', {
          framesCount: frames.length,
          serverConfidence: livenessResult.confidence
        });
      } catch (err) {
        // If server validation fails with network error, show specific message
        const networkMessage = 'Erro de conexão com o servidor. Verifique sua internet e tente novamente.';
        rekko.emit('liveness_failed', { reason: 'network_error', error: err });
        setResult({ mode: options.mode, verified: false, registered: false });
        setErrorMessage(networkMessage);
        rekko.setState('result');
        return;
      }
    } else {
      rekko.emit('liveness_success', { framesCount: frames.length });
    }

    processImage(imageToProcess);
  };

  const handleLivenessFailed = () => {
    rekko.emit('liveness_failed');
    setResult({ mode: options.mode, verified: false, registered: false });
    setErrorMessage(texts.liveness.failed);
    rekko.setState('result');
  };

  const isSuccess = () => {
    if (!result) return false;
    if (result.mode === 'verify') return result.verified === true;
    if (result.mode === 'identify') return result.identified === true;
    return result.registered === true;
  };

  const handleSuccessClose = () => {
    if (result && isSuccess()) {
      rekko.handleSuccess(result);
    } else {
      handleClose();
    }
  };

  const renderScreen = () => {
    switch (state) {
      case 'consent':
        return (
          <ConsentScreen
            texts={texts.consent}
            privacyPolicyUrl={options.consent?.privacyPolicyUrl}
            onAccept={() => {
              rekko.emit('consent_accepted');
              rekko.setState('orientation');
            }}
            onDecline={() => {
              rekko.emit('consent_declined');
              rekko.close();
            }}
          />
        );

      case 'orientation':
        return (
          <OrientationScreen
            texts={texts.orientation}
            onContinue={() => {
              rekko.setState('camera');
            }}
          />
        );

      case 'camera':
        return (
          <CameraScreen
            texts={texts.camera}
            errorTexts={texts.errors}
            onCapture={handleCameraCapture}
            onError={(error) => {
              rekko.handleError(error);
            }}
            onEvent={(type, data) => {
              rekko.emit(type, data);
            }}
            autoCapture={sessionReady}
          />
        );

      case 'liveness':
        return (
          <LivenessScreen
            onComplete={handleLivenessComplete}
            onFailed={handleLivenessFailed}
            onSkip={() => {
              if (capturedImage) {
                processImage(capturedImage);
              }
            }}
          />
        );

      case 'processing':
        return <ProcessingScreen texts={texts.processing} />;

      case 'result':
        return (
          <ResultScreen
            texts={texts.result}
            success={isSuccess()}
            errorMessage={errorMessage || undefined}
            onRetry={() => {
              setErrorMessage(null);
              rekko.setState('camera');
            }}
            onClose={handleSuccessClose}
          />
        );

      default:
        return null;
    }
  };

  return (
    <Modal onClose={handleClose} logo={config.logo}>
      {renderScreen()}
    </Modal>
  );
}
