/**
 * OrientationItem Component
 * Instruction card with differentiated icons
 */

import type { ComponentChildren } from 'preact';
import styles from './OrientationScreen.module.css';

export interface OrientationItemProps {
  icon: 'neutral' | 'visible' | 'lighting' | 'framing';
  text: string;
}

type IconType = 'neutral' | 'visible' | 'lighting' | 'framing';

export function OrientationItem({ icon, text }: OrientationItemProps) {
  return (
    <div class={styles.card}>
      <div class={`${styles.cardIcon} ${getIconStyle(icon)}`}>
        {renderIcon(icon)}
      </div>
      <span class={styles.cardText}>{text}</span>
    </div>
  );
}

function getIconStyle(icon: IconType): string {
  switch (icon) {
    case 'neutral': return styles.iconNeutral ?? '';
    case 'visible': return styles.iconVisible ?? '';
    case 'lighting': return styles.iconLighting ?? '';
    case 'framing': return styles.iconFraming ?? '';
  }
}

function renderIcon(icon: IconType): ComponentChildren {
  switch (icon) {
    case 'neutral': return <NeutralIcon />;
    case 'visible': return <VisibleIcon />;
    case 'lighting': return <LightingIcon />;
    case 'framing': return <FramingIcon />;
  }
}

function NeutralIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="10" />
      <path d="M8 14s1.5 2 4 2 4-2 4-2" />
      <line x1="9" y1="9" x2="9.01" y2="9" />
      <line x1="15" y1="9" x2="15.01" y2="9" />
    </svg>
  );
}

function VisibleIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  );
}

function LightingIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="5" />
      <line x1="12" y1="1" x2="12" y2="3" />
      <line x1="12" y1="21" x2="12" y2="23" />
      <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
      <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
      <line x1="1" y1="12" x2="3" y2="12" />
      <line x1="21" y1="12" x2="23" y2="12" />
      <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
      <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
    </svg>
  );
}

function FramingIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="10" />
      <circle cx="12" cy="12" r="6" />
      <circle cx="12" cy="12" r="2" />
    </svg>
  );
}

export default OrientationItem;
