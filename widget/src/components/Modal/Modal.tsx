import type { ComponentChildren } from 'preact';
import styles from './Modal.module.css';

interface ModalProps {
  children: ComponentChildren;
  onClose: () => void;
  logo?: string;
}

export function Modal({ children, onClose, logo }: ModalProps) {
  const handleBackdropClick = (e: MouseEvent) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  const handleKeyDown = (e: KeyboardEvent) => {
    if (e.key === 'Escape') {
      onClose();
    }
  };

  return (
    <div
      class={styles.backdrop}
      onClick={handleBackdropClick}
      onKeyDown={handleKeyDown}
      role="dialog"
      aria-modal="true"
      tabIndex={-1}
    >
      <div class={styles.modal}>
        <button class={styles.closeButton} onClick={onClose} aria-label="Close">
          <CloseIcon />
        </button>

        {logo && (
          <div class={styles.logoContainer}>
            <img src={logo} alt="" class={styles.logo} />
          </div>
        )}

        <div class={styles.content}>{children}</div>
      </div>
    </div>
  );
}

function CloseIcon() {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M18 6L6 18M6 6l12 12" />
    </svg>
  );
}
