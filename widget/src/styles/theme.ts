import type { RekkoTheme } from '@/types';

const CSS_VAR_MAP: Record<keyof RekkoTheme, string> = {
  primaryColor: '--rekko-primary-color',
  primaryHoverColor: '--rekko-primary-hover-color',
  backgroundColor: '--rekko-background-color',
  surfaceColor: '--rekko-surface-color',
  textColor: '--rekko-text-color',
  textSecondaryColor: '--rekko-text-secondary-color',
  borderColor: '--rekko-border-color',
  borderRadius: '--rekko-border-radius',
  fontFamily: '--rekko-font-family',
};

export function applyTheme(theme: RekkoTheme | undefined, container: HTMLElement): void {
  if (!theme) return;

  Object.entries(theme).forEach(([key, value]) => {
    const cssVar = CSS_VAR_MAP[key as keyof RekkoTheme];
    if (cssVar && value) {
      container.style.setProperty(cssVar, value);
    }
  });
}

export function resetTheme(container: HTMLElement): void {
  Object.values(CSS_VAR_MAP).forEach((cssVar) => {
    container.style.removeProperty(cssVar);
  });
}
