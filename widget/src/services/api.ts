import type { RekkoError, RekkoMode } from '@/types';
import { Errors, fromApiError, isRekkoError } from '@/errors';

export interface SessionResponse {
  sessionId: string;
  expiresAt: string;
}

// API response types (snake_case from Go backend)
interface ApiVerifyResponse {
  verified: boolean;
  confidence: number;
  verification_id: string;
  latency_ms: number;
}

interface ApiRegisterResponse {
  face_id: string;
  external_id: string;
  quality_score: number;
  created_at: string;
}

// API response types for liveness (snake_case from Go backend)
interface ApiLivenessResponse {
  is_live: boolean;
  confidence: number;
  checks: {
    eyes_open: boolean;
    facing_camera: boolean;
    quality_ok: boolean;
    single_face: boolean;
  };
  reasons?: string[];
}

// Frontend types (camelCase)
export interface VerifyResponse {
  verified: boolean;
  confidence: number;
  faceId?: string;
}

export interface RegisterResponse {
  faceId: string;
  registered: boolean;
  qualityScore: number;
}

export interface LivenessResponse {
  isLive: boolean;
  confidence: number;
  checks: {
    eyesOpen: boolean;
    facingCamera: boolean;
    qualityOk: boolean;
    singleFace: boolean;
  };
  reasons?: string[];
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
    const response = await this.request<{ session_id: string; expires_at: string }>('/v1/widget/session', {
      method: 'POST',
      body: JSON.stringify({
        public_key: this.publicKey,
        origin: window.location.origin,
      }),
    });

    this.sessionId = response.session_id;
    return { sessionId: response.session_id, expiresAt: response.expires_at };
  }

  async verify(externalId: string, imageBase64: string): Promise<VerifyResponse> {
    if (!this.sessionId) {
      throw Errors.sessionExpired();
    }

    const formData = this.createFormData(externalId, imageBase64);
    const response = await this.requestMultipart<ApiVerifyResponse>('/v1/widget/verify', formData);

    // Map snake_case to camelCase
    return {
      verified: response.verified,
      confidence: response.confidence,
      faceId: response.verification_id,
    };
  }

  async register(externalId: string, imageBase64: string): Promise<RegisterResponse> {
    if (!this.sessionId) {
      throw Errors.sessionExpired();
    }

    const formData = this.createFormData(externalId, imageBase64);
    const response = await this.requestMultipart<ApiRegisterResponse>('/v1/widget/register', formData);

    // Map snake_case to camelCase - if we got face_id, registration succeeded
    return {
      faceId: response.face_id,
      registered: !!response.face_id,
      qualityScore: response.quality_score,
    };
  }

  async validateLiveness(imageBase64: string): Promise<LivenessResponse> {
    if (!this.sessionId) {
      throw Errors.sessionExpired();
    }

    const formData = this.createLivenessFormData(imageBase64);
    const response = await this.requestMultipart<ApiLivenessResponse>('/v1/widget/validate', formData);

    // Map snake_case to camelCase
    return {
      isLive: response.is_live,
      confidence: response.confidence,
      checks: {
        eyesOpen: response.checks.eyes_open,
        facingCamera: response.checks.facing_camera,
        qualityOk: response.checks.quality_ok,
        singleFace: response.checks.single_face,
      },
      reasons: response.reasons,
    };
  }

  private createLivenessFormData(imageBase64: string): FormData {
    const formData = new FormData();
    formData.append('session_id', this.sessionId!);

    // Convert base64 to blob
    const base64Data = imageBase64.replace(/^data:image\/\w+;base64,/, '');
    const binaryData = atob(base64Data);
    const bytes = new Uint8Array(binaryData.length);
    for (let i = 0; i < binaryData.length; i++) {
      bytes[i] = binaryData.charCodeAt(i);
    }
    const blob = new Blob([bytes], { type: 'image/jpeg' });
    formData.append('image', blob, 'liveness.jpg');

    return formData;
  }

  private createFormData(externalId: string, imageBase64: string): FormData {
    const formData = new FormData();
    formData.append('session_id', this.sessionId!);
    formData.append('external_id', externalId);

    // Convert base64 to blob
    const base64Data = imageBase64.replace(/^data:image\/\w+;base64,/, '');
    const binaryData = atob(base64Data);
    const bytes = new Uint8Array(binaryData.length);
    for (let i = 0; i < binaryData.length; i++) {
      bytes[i] = binaryData.charCodeAt(i);
    }
    const blob = new Blob([bytes], { type: 'image/jpeg' });
    formData.append('image', blob, 'capture.jpg');

    return formData;
  }

  private async requestMultipart<T>(path: string, formData: FormData): Promise<T> {
    const url = `${this.baseUrl}${path}`;

    try {
      const response = await fetch(url, {
        method: 'POST',
        body: formData,
        // Don't set Content-Type header - browser will set it with boundary
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
      const errorData = data.error || data;
      const code = errorData.code;
      const message = errorData.message;

      // Use the fromApiError function to get friendly messages
      if (code) {
        return fromApiError(code, message);
      }

      return Errors.unknown(data);
    } catch {
      return Errors.networkError(`HTTP ${response.status}`);
    }
  }
}
