import type { LocaleTexts } from '@/types';
import styles from './ProcessingScreen.module.css';

interface ProcessingScreenProps {
  texts: LocaleTexts['processing'];
}

export function ProcessingScreen({ texts }: ProcessingScreenProps) {
  return (
    <div class={styles.container}>
      <div class={styles.spinnerContainer}>
        <svg class={styles.spinner} viewBox="0 0 50 50">
          <circle
            class={styles.track}
            cx="25"
            cy="25"
            r="20"
            fill="none"
            strokeWidth="4"
          />
          <circle
            class={styles.progress}
            cx="25"
            cy="25"
            r="20"
            fill="none"
            strokeWidth="4"
            strokeLinecap="round"
          />
        </svg>
      </div>

      <h2 class={styles.title}>{texts.title}</h2>
      <p class={styles.message}>{texts.message}</p>
    </div>
  );
}
