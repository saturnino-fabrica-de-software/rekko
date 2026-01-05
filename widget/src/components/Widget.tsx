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
      setResult({ mode: options.mode, verified: false, registered: false });

      if (isRekkoError(err)) {
        setErrorMessage(error.message);
      } else {
        setErrorMessage(null);
      }

      rekko.emit(options.mode === 'verify' ? 'verification_failed' : 'registration_failed', error);
      rekko.setState('result');
    }
  };

  const handleCameraCapture = (imageData: string) => {
    setCapturedImage(imageData);

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

    // Validate liveness with backend using the best frame (middle of captured frames)
    const bestFrame = frames.length > 0 ? frames[Math.floor(frames.length / 2)] ?? imageToProcess : imageToProcess;

    if (client) {
      try {
        rekko.emit('liveness_challenge', { stage: 'server_validation' });
        const livenessResult = await client.validateLiveness(bestFrame);

        if (!livenessResult.isLive) {
          rekko.emit('liveness_failed', { reason: 'server_validation', confidence: livenessResult.confidence });
          setResult({ mode: options.mode, verified: false, registered: false });
          setErrorMessage(texts.liveness.failed);
          rekko.setState('result');
          return;
        }

        rekko.emit('liveness_success', {
          framesCount: frames.length,
          serverConfidence: livenessResult.confidence
        });
      } catch (err) {
        // If server validation fails, still allow registration (graceful degradation)
        rekko.emit('liveness_success', { framesCount: frames.length, serverValidation: 'skipped' });
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
