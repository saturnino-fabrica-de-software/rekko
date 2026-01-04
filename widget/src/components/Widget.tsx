import { Modal } from './Modal/Modal';
import { ConsentScreen } from './ConsentScreen/ConsentScreen';
import { CameraScreen } from './CameraScreen/CameraScreen';
import { ProcessingScreen } from './ProcessingScreen/ProcessingScreen';
import { ResultScreen } from './ResultScreen/ResultScreen';
import type { RekkoWidget } from '@/Rekko';

interface WidgetProps {
  rekko: RekkoWidget;
}

export function Widget({ rekko }: WidgetProps) {
  const state = rekko.getState();
  const texts = rekko.getTexts();

  if (!texts) return null;

  const handleClose = () => {
    rekko.close();
  };

  const renderScreen = () => {
    switch (state) {
      case 'consent':
        return (
          <ConsentScreen
            texts={texts.consent}
            onAccept={() => {
              rekko.emit('consent_accepted');
              rekko.setState('camera');
            }}
            onDecline={() => {
              rekko.emit('consent_declined');
              rekko.close();
            }}
          />
        );

      case 'camera':
        return (
          <CameraScreen
            texts={texts.camera}
            errorTexts={texts.errors}
            onCapture={(_imageData: string) => {
              rekko.emit('capture_started');
              rekko.setState('processing');
              // TODO: Send to API
            }}
            onError={(error) => {
              rekko.handleError(error);
            }}
            onEvent={(type, data) => {
              rekko.emit(type, data);
            }}
          />
        );

      case 'processing':
        return <ProcessingScreen texts={texts.processing} />;

      case 'result':
        return (
          <ResultScreen
            texts={texts.result}
            success={false} // TODO: Get from state
            onRetry={() => rekko.setState('camera')}
            onClose={handleClose}
          />
        );

      default:
        return null;
    }
  };

  return (
    <Modal onClose={handleClose} logo={rekko.getConfig()?.logo}>
      {renderScreen()}
    </Modal>
  );
}
