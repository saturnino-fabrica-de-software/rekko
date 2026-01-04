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

type WidgetState = 'idle' | 'consent' | 'camera' | 'processing' | 'result';

class RekkoWidget implements RekkoInstance {
  private config: RekkoConfig | null = null;
  private options: RekkoOpenOptions | null = null;
  private container: HTMLElement | null = null;
  private state: WidgetState = 'idle';
  private initialized = false;

  init(config: RekkoConfig): void {
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
    this.initialized = true;
  }

  open(options: RekkoOpenOptions): void {
    if (!this.initialized || !this.config) {
      throw new Error('Rekko: call init() before open()');
    }
    if (!options.onSuccess || !options.onError) {
      throw new Error('Rekko: onSuccess and onError callbacks are required');
    }
    if (options.mode === 'verify' && !options.externalId) {
      throw new Error('Rekko: externalId is required for verify mode');
    }

    this.options = options;
    this.createContainer();
    this.emit('widget_opened');
    this.setState('consent');
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
