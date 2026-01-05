/**
 * OrientationItem Component
 * Minimal instruction item with check icon
 */

import styles from './OrientationScreen.module.css';

export interface OrientationItemProps {
  icon: 'neutral' | 'visible' | 'lighting' | 'framing';
  text: string;
}

/**
 * Single orientation instruction item - minimal design
 */
export function OrientationItem({ text }: OrientationItemProps) {
  return (
    <div class={styles.item}>
      <div class={styles.checkIcon}>
        <CheckIcon />
      </div>
      <span class={styles.itemText}>{text}</span>
    </div>
  );
}

function CheckIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 18 18" fill="none">
      <circle cx="9" cy="9" r="9" fill="currentColor" opacity="0.1" />
      <path
        d="M5.5 9L8 11.5L12.5 6.5"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

export default OrientationItem;
