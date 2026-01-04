import type { RekkoError, RekkoMode } from '@/types';
import { Errors } from '@/errors';

export interface SessionResponse {
  sessionId: string;
  expiresAt: string;
}

export interface VerifyRequest {
  sessionId: string;
  externalId: string;
  image: string;
}

export interface VerifyResponse {
  verified: boolean;
  confidence: number;
  faceId?: string;
}

export interface RegisterRequest {
  sessionId: string;
  externalId: string;
  image: string;
}

export interface RegisterResponse {
  faceId: string;
  registered: boolean;
  qualityScore: number;
}

export class ApiClient {
  private baseUrl: string;
  private publicKey: string;
  private sessionId: string | null = null;

  constructor(baseUrl: string, publicKey: string) {
    this.baseUrl = baseUrl.replace(/\/$/, '');
    this.publicKey = publicKey;
  }

  async createSession(): Promise<SessionResponse> {
    const response = await this.request<SessionResponse>('/v1/widget/session', {
      method: 'POST',
      body: JSON.stringify({
        publicKey: this.publicKey,
        origin: window.location.origin,
      }),
    });

    this.sessionId = response.sessionId;
    return response;
  }

  async verify(externalId: string, imageBase64: string): Promise<VerifyResponse> {
    if (!this.sessionId) {
      throw Errors.sessionExpired();
    }

    return this.request<VerifyResponse>('/v1/widget/verify', {
      method: 'POST',
      body: JSON.stringify({
        sessionId: this.sessionId,
        externalId,
        image: imageBase64,
      }),
    });
  }

  async register(externalId: string, imageBase64: string): Promise<RegisterResponse> {
    if (!this.sessionId) {
      throw Errors.sessionExpired();
    }

    return this.request<RegisterResponse>('/v1/widget/register', {
      method: 'POST',
      body: JSON.stringify({
        sessionId: this.sessionId,
        externalId,
        image: imageBase64,
      }),
    });
  }

  async process(
    mode: RekkoMode,
    externalId: string,
    imageBase64: string
  ): Promise<VerifyResponse | RegisterResponse> {
    if (mode === 'verify') {
      return this.verify(externalId, imageBase64);
    }
    return this.register(externalId, imageBase64);
  }

  private async request<T>(path: string, options: RequestInit): Promise<T> {
    const url = `${this.baseUrl}${path}`;

    try {
      const response = await fetch(url, {
        ...options,
        headers: {
          'Content-Type': 'application/json',
          ...options.headers,
        },
      });

      if (!response.ok) {
        const error = await this.parseErrorResponse(response);
        throw error;
      }

      return response.json() as Promise<T>;
    } catch (err) {
      if (isRekkoError(err)) {
        throw err;
      }

      if (err instanceof TypeError && err.message.includes('fetch')) {
        throw Errors.networkError(err);
      }

      throw Errors.unknown(err);
    }
  }

  private async parseErrorResponse(response: Response): Promise<RekkoError> {
    try {
      const data = await response.json();

      switch (response.status) {
        case 403:
          if (data.code === 'DOMAIN_NOT_ALLOWED') {
            return Errors.domainNotAllowed();
          }
          if (data.code === 'INVALID_PUBLIC_KEY') {
            return Errors.invalidPublicKey();
          }
          break;
        case 401:
          return Errors.sessionExpired();
        case 400:
          if (data.code === 'NO_FACE_DETECTED') {
            return Errors.noFaceDetected();
          }
          if (data.code === 'MULTIPLE_FACES') {
            return Errors.multipleFaces();
          }
          if (data.code === 'LOW_QUALITY') {
            return Errors.lowQuality();
          }
          break;
        case 404:
          return Errors.faceNotFound();
      }

      return Errors.unknown(data);
    } catch {
      return Errors.networkError(`HTTP ${response.status}`);
    }
  }
}

function isRekkoError(err: unknown): err is RekkoError {
  return (
    typeof err === 'object' &&
    err !== null &&
    'code' in err &&
    'message' in err
  );
}
