/**
 * ConsentScreen Component
 * Clean, professional consent UI with LGPD compliance elements
 */

import type { LocaleTexts } from '@/types';
import styles from './ConsentScreen.module.css';

interface ConsentScreenProps {
  texts: LocaleTexts['consent'];
  privacyPolicyUrl?: string;
  onAccept: () => void;
  onDecline: () => void;
}

export function ConsentScreen({ texts, privacyPolicyUrl, onAccept, onDecline }: ConsentScreenProps) {
  return (
    <div class={styles.container}>
      <div class={styles.content}>
        {/* Icon with subtle glow effect */}
        <div class={styles.iconWrapper}>
          <div class={styles.iconGlow} />
          <div class={styles.iconContainer}>
            <FaceIdIcon />
          </div>
        </div>

        <h2 class={styles.title}>{texts.title}</h2>
        <p class={styles.body}>{texts.body}</p>

        {/* Privacy Policy Link */}
        {privacyPolicyUrl && (
          <a
            href={privacyPolicyUrl}
            target="_blank"
            rel="noopener noreferrer"
            class={styles.privacyLink}
          >
            <LockIcon />
            <span>Política de Privacidade</span>
            <ExternalLinkIcon />
          </a>
        )}
      </div>

      {/* Actions */}
      <div class={styles.actions}>
        <button class={styles.primaryButton} onClick={onAccept}>
          {texts.accept}
        </button>
        <button class={styles.secondaryButton} onClick={onDecline}>
          {texts.decline}
        </button>
      </div>

      {/* Security Badge */}
      <div class={styles.securityBadge}>
        <ShieldIcon />
        <span>Dados criptografados e protegidos</span>
      </div>
    </div>
  );
}

// Icons
function FaceIdIcon() {
  return (
    <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <path d="M4 8V6a2 2 0 0 1 2-2h2" />
      <path d="M4 16v2a2 2 0 0 0 2 2h2" />
      <path d="M16 4h2a2 2 0 0 1 2 2v2" />
      <path d="M16 20h2a2 2 0 0 0 2-2v-2" />
      <circle cx="12" cy="12" r="4" />
    </svg>
  );
}

function LockIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <rect width="18" height="11" x="3" y="11" rx="2" ry="2" />
      <path d="M7 11V7a5 5 0 0 1 10 0v4" />
    </svg>
  );
}

function ExternalLinkIcon() {
  return (
    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6" />
      <polyline points="15 3 21 3 21 9" />
      <line x1="10" x2="21" y1="14" y2="3" />
    </svg>
  );
}

function ShieldIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
      <path d="m9 12 2 2 4-4" />
    </svg>
  );
}

export default ConsentScreen;
