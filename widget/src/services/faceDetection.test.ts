/**
 * Face Detection Service Tests
 * Unit tests for faceDetection.ts
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import {
  loadModels,
  areModelsLoaded,
  getLoadingState,
  resetService,
} from './faceDetection';

// Mock face-api.js
vi.mock('@vladmandic/face-api', () => ({
  nets: {
    tinyFaceDetector: {
      loadFromUri: vi.fn().mockResolvedValue(undefined),
    },
    faceLandmark68TinyNet: {
      loadFromUri: vi.fn().mockResolvedValue(undefined),
    },
    faceExpressionNet: {
      loadFromUri: vi.fn().mockResolvedValue(undefined),
    },
  },
  TinyFaceDetectorOptions: vi.fn(),
  detectAllFaces: vi.fn().mockReturnValue({
    withFaceLandmarks: vi.fn().mockResolvedValue([]),
  }),
}));

describe('faceDetection service', () => {
  beforeEach(() => {
    resetService();
    vi.clearAllMocks();
  });

  describe('getLoadingState', () => {
    it('should return initial state when not loaded', () => {
      const state = getLoadingState();

      expect(state.isLoading).toBe(false);
      expect(state.isLoaded).toBe(false);
      expect(state.error).toBeNull();
      expect(state.progress).toBe(0);
      expect(state.loadedModels).toEqual([]);
    });
  });

  describe('areModelsLoaded', () => {
    it('should return false when models not loaded', () => {
      expect(areModelsLoaded()).toBe(false);
    });

    it('should return true after loading models', async () => {
      await loadModels();
      expect(areModelsLoaded()).toBe(true);
    });
  });

  describe('loadModels', () => {
    it('should load models successfully', async () => {
      await loadModels();

      const state = getLoadingState();
      expect(state.isLoaded).toBe(true);
      expect(state.isLoading).toBe(false);
      expect(state.progress).toBe(100);
      expect(state.error).toBeNull();
    });

    it('should call onProgress callback during loading', async () => {
      const onProgress = vi.fn();

      await loadModels(undefined, onProgress);

      expect(onProgress).toHaveBeenCalled();
      expect(onProgress).toHaveBeenLastCalledWith(100);
    });

    it('should not reload models if already loaded', async () => {
      const faceapi = await import('@vladmandic/face-api');

      await loadModels();
      await loadModels();

      // Should only load once
      expect(faceapi.nets.tinyFaceDetector.loadFromUri).toHaveBeenCalledTimes(1);
    });

    it('should load specific models when provided', async () => {
      const faceapi = await import('@vladmandic/face-api');

      await loadModels(['tiny_face_detector']);

      expect(faceapi.nets.tinyFaceDetector.loadFromUri).toHaveBeenCalled();
      expect(faceapi.nets.faceLandmark68TinyNet.loadFromUri).not.toHaveBeenCalled();
    });

    it('should handle concurrent loading calls', async () => {
      const faceapi = await import('@vladmandic/face-api');

      // Call loadModels twice concurrently
      const [result1, result2] = await Promise.all([loadModels(), loadModels()]);

      expect(result1).toBeUndefined();
      expect(result2).toBeUndefined();
      expect(faceapi.nets.tinyFaceDetector.loadFromUri).toHaveBeenCalledTimes(1);
    });
  });

  describe('resetService', () => {
    it('should reset all state', async () => {
      await loadModels();
      expect(areModelsLoaded()).toBe(true);

      resetService();

      expect(areModelsLoaded()).toBe(false);
      const state = getLoadingState();
      expect(state.isLoaded).toBe(false);
      expect(state.progress).toBe(0);
    });
  });
});

describe('faceDetection with error handling', () => {
  beforeEach(() => {
    resetService();
  });

  it('should handle model loading failure', async () => {
    const faceapi = await import('@vladmandic/face-api');
    const loadError = new Error('Network error');

    vi.mocked(faceapi.nets.tinyFaceDetector.loadFromUri).mockRejectedValueOnce(loadError);
    vi.mocked(faceapi.nets.tinyFaceDetector.loadFromUri).mockRejectedValueOnce(loadError);

    await expect(loadModels(['tiny_face_detector'])).rejects.toThrow();

    const state = getLoadingState();
    expect(state.isLoaded).toBe(false);
    expect(state.error).not.toBeNull();
  });
});
