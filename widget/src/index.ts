import './styles/variables.css';
import { rekkoInstance } from './Rekko';
import type { RekkoInstance, RekkoConfig, RekkoOpenOptions, RekkoResult, RekkoError, RekkoEvent } from './types';

export type { RekkoInstance, RekkoConfig, RekkoOpenOptions, RekkoResult, RekkoError, RekkoEvent };

export const Rekko: RekkoInstance = {
  init: (config: RekkoConfig) => rekkoInstance.init(config),
  open: (options: RekkoOpenOptions) => rekkoInstance.open(options),
  close: () => rekkoInstance.close(),
  isInitialized: () => rekkoInstance.isInitialized(),
};

if (typeof window !== 'undefined') {
  window.Rekko = Rekko;
}
