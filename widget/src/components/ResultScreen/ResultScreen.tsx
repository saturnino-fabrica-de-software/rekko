import type { LocaleTexts } from '@/types';
import styles from './ResultScreen.module.css';

interface ResultScreenProps {
  texts: LocaleTexts['result'];
  success: boolean;
  errorMessage?: string;
  onRetry: () => void;
  onClose: () => void;
}

export function ResultScreen({ texts, success, errorMessage, onRetry, onClose }: ResultScreenProps) {
  const displayMessage = success
    ? texts.successMessage
    : errorMessage || texts.errorMessage;

  return (
    <div class={styles.container}>
      <div class={`${styles.icon} ${success ? styles.success : styles.error}`}>
        {success ? <CheckIcon /> : <XIcon />}
      </div>

      <h2 class={styles.title}>
        {success ? texts.successTitle : texts.errorTitle}
      </h2>
      <p class={styles.message}>
        {displayMessage}
      </p>

      <div class={styles.buttons}>
        {success ? (
          <button class={styles.primaryButton} onClick={onClose}>
            {texts.close}
          </button>
        ) : (
          <>
            <button class={styles.primaryButton} onClick={onRetry}>
              {texts.retry}
            </button>
            <button class={styles.secondaryButton} onClick={onClose}>
              {texts.close}
            </button>
          </>
        )}
      </div>
    </div>
  );
}

function CheckIcon() {
  return (
    <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
      <path d="M20 6L9 17l-5-5" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

function XIcon() {
  return (
    <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
      <path d="M18 6L6 18M6 6l12 12" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}
