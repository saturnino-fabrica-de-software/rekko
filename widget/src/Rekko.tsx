import { render } from 'preact';
import type {
  RekkoConfig,
  RekkoOpenOptions,
  RekkoInstance,
  RekkoEvent,
  RekkoEventType,
  LocaleTexts,
} from '@/types';
import { applyTheme, resetTheme } from '@/styles/theme';

type WidgetState = 'idle' | 'consent' | 'orientation' | 'camera' | 'liveness' | 'processing' | 'result';

interface WidgetSession {
  sessionId: string;
  expiresAt: Date;
}

class RekkoWidget implements RekkoInstance {
  private config: RekkoConfig | null = null;
  private options: RekkoOpenOptions | null = null;
  private container: HTMLElement | null = null;
  private state: WidgetState = 'idle';
  private initialized = false;
  private session: WidgetSession | null = null;
  private initError: string | null = null;

  async init(config: RekkoConfig): Promise<void> {
    if (!config.publicKey) {
      throw new Error('Rekko: publicKey is required');
    }
    if (!config.locale) {
      throw new Error('Rekko: locale is required');
    }
    if (!config.texts || Object.keys(config.texts).length === 0) {
      throw new Error('Rekko: texts are required');
    }
    if (!config.texts[config.locale]) {
      throw new Error(`Rekko: texts for locale "${config.locale}" not found`);
    }

    this.config = {
      apiUrl: 'https://api.rekko.io',
      ...config,
    };

    // Validate pk_live and create session
    try {
      await this.createSession();
      this.initialized = true;
      this.initError = null;
    } catch (err) {
      this.initError = err instanceof Error ? err.message : 'Unknown error';
      throw err;
    }
  }

  private async createSession(): Promise<void> {
    if (!this.config) {
      throw new Error('Rekko: config not set');
    }

    const response = await fetch(`${this.config.apiUrl}/v1/widget/session`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        public_key: this.config.publicKey,
        origin: window.location.origin,
      }),
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: { message: 'Unknown error' } }));
      throw new Error(`Rekko: ${error.error?.message || 'Invalid public key'}`);
    }

    const data = await response.json();
    this.session = {
      sessionId: data.session_id,
      expiresAt: new Date(data.expires_at),
    };
  }

  private isSessionExpired(): boolean {
    if (!this.session) return true;
    // Add 30 seconds buffer to avoid edge cases
    return new Date() >= new Date(this.session.expiresAt.getTime() - 30000);
  }

  getSessionId(): string | null {
    return this.session?.sessionId ?? null;
  }

  async open(options: RekkoOpenOptions): Promise<void> {
    // Validate callbacks first (required for error handling)
    if (!options.onSuccess || !options.onError) {
      console.error('Rekko: onSuccess and onError callbacks are required');
      return;
    }

    // Check if init was successful
    if (!this.initialized || !this.config) {
      const message = this.initError || 'call init() before open()';
      options.onError({ code: 'INVALID_PUBLIC_KEY', message });
      return;
    }

    if (options.mode === 'verify' && !options.externalId) {
      options.onError({ code: 'UNKNOWN_ERROR', message: 'externalId is required for verify mode' });
      return;
    }

    // Renew session if expired
    try {
      if (this.isSessionExpired()) {
        await this.createSession();
      }
    } catch (err) {
      options.onError({ code: 'SESSION_EXPIRED', message: 'Failed to renew session' });
      return;
    }

    this.options = options;
    this.createContainer();
    this.emit('widget_opened');

    // Determine starting screen based on skip options
    let startScreen: WidgetState = 'consent';
    if (options.skipConsent) {
      startScreen = options.skipOrientation ? 'camera' : 'orientation';
    }
    this.setState(startScreen);
  }

  close(): void {
    if (this.container) {
      this.emit('widget_closed');
      resetTheme(this.container);
      render(null, this.container);
      this.container.remove();
      this.container = null;
    }
    this.options = null;
    this.state = 'idle';
  }

  isInitialized(): boolean {
    return this.initialized;
  }

  version(): string {
    return '0.2.0';
  }

  getConfig(): RekkoConfig | null {
    return this.config;
  }

  getOptions(): RekkoOpenOptions | null {
    return this.options;
  }

  getTexts(): LocaleTexts | null {
    if (!this.config) return null;
    return this.config.texts[this.config.locale] ?? null;
  }

  getState(): WidgetState {
    return this.state;
  }

  setState(state: WidgetState): void {
    this.state = state;
    this.render();
  }

  emit(type: RekkoEventType, data?: unknown): void {
    if (!this.options?.onEvent) return;

    const event: RekkoEvent = {
      type,
      timestamp: Date.now(),
      data,
    };

    try {
      this.options.onEvent(event);
    } catch (err) {
      console.error('Rekko: onEvent callback error:', err);
    }
  }

  handleSuccess(result: Parameters<RekkoOpenOptions['onSuccess']>[0]): void {
    if (!this.options) return;

    try {
      this.options.onSuccess(result);
    } catch (err) {
      console.error('Rekko: onSuccess callback error:', err);
    }

    this.close();
  }

  handleError(error: Parameters<RekkoOpenOptions['onError']>[0]): void {
    if (!this.options) return;

    try {
      this.options.onError(error);
    } catch (err) {
      console.error('Rekko: onError callback error:', err);
    }

    this.close();
  }

  private createContainer(): void {
    this.container = document.createElement('div');
    this.container.id = 'rekko-widget-container';
    this.container.className = 'rekko-widget';
    document.body.appendChild(this.container);

    if (this.config?.theme) {
      applyTheme(this.config.theme, this.container);
    }
  }

  private render(): void {
    if (!this.container) return;

    import('./components/Widget').then(({ Widget }) => {
      if (this.container) {
        render(<Widget rekko={this} />, this.container);
      }
    });
  }
}

export const rekkoInstance = new RekkoWidget();
export type { RekkoWidget };
