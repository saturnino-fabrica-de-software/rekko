import type { LocaleTexts } from '@/types';
import styles from './ConsentScreen.module.css';

interface ConsentScreenProps {
  texts: LocaleTexts['consent'];
  onAccept: () => void;
  onDecline: () => void;
}

export function ConsentScreen({ texts, onAccept, onDecline }: ConsentScreenProps) {
  return (
    <div class={styles.container}>
      <div class={styles.icon}>
        <ShieldIcon />
      </div>

      <h2 class={styles.title}>{texts.title}</h2>
      <p class={styles.body}>{texts.body}</p>

      <div class={styles.buttons}>
        <button class={styles.primaryButton} onClick={onAccept}>
          {texts.accept}
        </button>
        <button class={styles.secondaryButton} onClick={onDecline}>
          {texts.decline}
        </button>
      </div>
    </div>
  );
}

function ShieldIcon() {
  return (
    <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
      <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
      <path d="M9 12l2 2 4-4" />
    </svg>
  );
}
