/**
 * Mock for @vladmandic/face-api
 * Provides lightweight mock implementation for testing
 */

export const nets = {
  tinyFaceDetector: {
    loadFromUri: () => Promise.resolve(),
  },
  faceLandmark68TinyNet: {
    loadFromUri: () => Promise.resolve(),
  },
  faceExpressionNet: {
    loadFromUri: () => Promise.resolve(),
  },
};

export class TinyFaceDetectorOptions {
  constructor(_options?: { inputSize?: number; scoreThreshold?: number }) {}
}

export const detectAllFaces = () => ({
  withFaceLandmarks: () => Promise.resolve([]),
});
