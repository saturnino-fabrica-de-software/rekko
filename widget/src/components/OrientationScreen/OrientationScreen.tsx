/**
 * OrientationScreen Component
 * Pre-camera instructions screen with clean, minimal design
 */

import type { LocaleTexts } from '@/types';
import { OrientationItem } from './OrientationItem';
import styles from './OrientationScreen.module.css';

interface OrientationScreenProps {
  texts: LocaleTexts['orientation'];
  onContinue: () => void;
}

/**
 * Orientation screen shown before camera access
 * Displays instructions for optimal face capture
 */
export function OrientationScreen({ texts, onContinue }: OrientationScreenProps) {
  return (
    <div class={styles.container}>
      <div class={styles.content}>
        <h2 class={styles.title}>{texts.title}</h2>
        <p class={styles.subtitle}>{texts.subtitle}</p>

        <div class={styles.illustration}>
          <FaceIllustration />
        </div>

        <div class={styles.instructions}>
          <OrientationItem icon="neutral" text={texts.instructions.neutral} />
          <OrientationItem icon="visible" text={texts.instructions.visible} />
          <OrientationItem icon="lighting" text={texts.instructions.lighting} />
          <OrientationItem icon="framing" text={texts.instructions.framing} />
        </div>
      </div>

      <button class={styles.continueButton} onClick={onContinue}>
        {texts.continue}
        <ArrowIcon />
      </button>
    </div>
  );
}

/**
 * Arrow icon for the continue button
 */
function ArrowIcon() {
  return (
    <svg class={styles.buttonArrow} viewBox="0 0 16 16" fill="none">
      <path
        d="M3 8h10M9 4l4 4-4 4"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

/**
 * Simplified face illustration
 */
function FaceIllustration() {
  return (
    <svg
      class={styles.faceIllustration}
      viewBox="0 0 140 160"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      {/* Outer dashed oval guide */}
      <ellipse
        cx="70"
        cy="80"
        rx="55"
        ry="70"
        stroke="var(--rekko-primary, #3b82f6)"
        strokeWidth="2"
        strokeDasharray="6 4"
        fill="none"
        opacity="0.4"
      />

      {/* Face fill */}
      <ellipse
        cx="70"
        cy="78"
        rx="38"
        ry="48"
        fill="var(--rekko-primary, #3b82f6)"
        opacity="0.08"
      />

      {/* Face outline */}
      <ellipse
        cx="70"
        cy="78"
        rx="38"
        ry="48"
        stroke="var(--rekko-primary, #3b82f6)"
        strokeWidth="2"
        fill="none"
      />

      {/* Left eye */}
      <ellipse cx="56" cy="70" rx="5" ry="3" fill="var(--rekko-primary, #3b82f6)" opacity="0.6" />

      {/* Right eye */}
      <ellipse cx="84" cy="70" rx="5" ry="3" fill="var(--rekko-primary, #3b82f6)" opacity="0.6" />

      {/* Nose hint */}
      <path
        d="M70 78 L70 90"
        stroke="var(--rekko-primary, #3b82f6)"
        strokeWidth="1.5"
        strokeLinecap="round"
        opacity="0.4"
      />

      {/* Smile */}
      <path
        d="M60 100 Q70 107 80 100"
        stroke="var(--rekko-primary, #3b82f6)"
        strokeWidth="1.5"
        strokeLinecap="round"
        fill="none"
        opacity="0.5"
      />

      {/* Success badge */}
      <circle cx="115" cy="35" r="14" fill="#10b981" />
      <path
        d="M108 35 L113 40 L122 31"
        stroke="white"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

export default OrientationScreen;
