/**
 * Face Detection Service
 * Manages face-api.js model loading and detection
 */

import * as faceapi from '@vladmandic/face-api';
import { MODEL_CONFIG, DEFAULT_DETECTION_CONFIG } from '../config/detection';
import type {
  FaceDetectionResult,
  FacePosition,
  FaceLandmarks,
  DetectionQuality,
  ModelLoadingState,
  FaceApiModel,
  Point,
} from '../types/faceDetection';

// Singleton state
let modelsLoaded = false;
let loadingPromise: Promise<void> | null = null;
let loadedModels: Set<FaceApiModel> = new Set();

/**
 * Model loading state for external monitoring
 */
let loadingState: ModelLoadingState = {
  isLoading: false,
  isLoaded: false,
  error: null,
  progress: 0,
  loadedModels: [],
};

/**
 * Get current loading state
 */
export function getLoadingState(): ModelLoadingState {
  return { ...loadingState };
}

/**
 * Load face detection models with lazy loading
 * Uses singleton pattern to avoid reloading
 */
export async function loadModels(
  models: FaceApiModel[] = MODEL_CONFIG.phase1Models as FaceApiModel[],
  onProgress?: (progress: number) => void
): Promise<void> {
  // Already loaded
  if (modelsLoaded && models.every(m => loadedModels.has(m))) {
    return;
  }

  // Already loading, wait for it
  if (loadingPromise) {
    return loadingPromise;
  }

  loadingState = {
    isLoading: true,
    isLoaded: false,
    error: null,
    progress: 0,
    loadedModels: [],
  };

  loadingPromise = (async () => {
    try {
      const totalModels = models.length;
      let loadedCount = 0;

      // Try CDN first, fall back to local
      const modelPath = MODEL_CONFIG.cdnBaseUrl;

      for (const model of models) {
        if (loadedModels.has(model)) {
          loadedCount++;
          continue;
        }

        try {
          await loadSingleModel(model, modelPath);
          loadedModels.add(model);
          loadedCount++;

          const progress = Math.round((loadedCount / totalModels) * 100);
          loadingState.progress = progress;
          loadingState.loadedModels = Array.from(loadedModels);
          onProgress?.(progress);
        } catch (cdnError) {
          // Try local fallback
          console.warn(`CDN load failed for ${model}, trying local fallback`);
          try {
            await loadSingleModel(model, MODEL_CONFIG.localFallbackPath);
            loadedModels.add(model);
            loadedCount++;

            const progress = Math.round((loadedCount / totalModels) * 100);
            loadingState.progress = progress;
            loadingState.loadedModels = Array.from(loadedModels);
            onProgress?.(progress);
          } catch (localError) {
            throw new Error(`Failed to load model ${model}: ${localError}`);
          }
        }
      }

      modelsLoaded = true;
      loadingState = {
        isLoading: false,
        isLoaded: true,
        error: null,
        progress: 100,
        loadedModels: Array.from(loadedModels),
      };
    } catch (error) {
      loadingState = {
        isLoading: false,
        isLoaded: false,
        error: error instanceof Error ? error : new Error(String(error)),
        progress: 0,
        loadedModels: Array.from(loadedModels),
      };
      loadingPromise = null;
      throw error;
    }
  })();

  return loadingPromise;
}

/**
 * Load a single model
 */
async function loadSingleModel(model: FaceApiModel, basePath: string): Promise<void> {
  switch (model) {
    case 'tiny_face_detector':
      await faceapi.nets.tinyFaceDetector.loadFromUri(basePath);
      break;
    case 'face_landmark_68_tiny':
      await faceapi.nets.faceLandmark68TinyNet.loadFromUri(basePath);
      break;
    case 'face_expression':
      await faceapi.nets.faceExpressionNet.loadFromUri(basePath);
      break;
    default:
      throw new Error(`Unknown model: ${model}`);
  }
}

/**
 * Check if models are loaded
 */
export function areModelsLoaded(): boolean {
  return modelsLoaded;
}

/**
 * Detect faces in a video element
 */
export async function detectFaces(
  video: HTMLVideoElement
): Promise<FaceDetectionResult> {
  if (!modelsLoaded) {
    throw new Error('Models not loaded. Call loadModels() first.');
  }

  const timestamp = Date.now();

  try {
    // Run detection with landmarks
    const detections = await faceapi
      .detectAllFaces(video, new faceapi.TinyFaceDetectorOptions({
        inputSize: 320, // Smaller for performance
        scoreThreshold: 0.5,
      }))
      .withFaceLandmarks(true); // Use tiny landmarks

    if (detections.length === 0) {
      return {
        detected: false,
        faceCount: 0,
        position: null,
        landmarks: null,
        quality: null,
        rawScore: 0,
        timestamp,
      };
    }

    // Get the most prominent face (highest score)
    const bestDetection = detections.reduce((best, current) =>
      current.detection.score > best.detection.score ? current : best
    );

    const box = bestDetection.detection.box;
    const landmarks = bestDetection.landmarks;
    const score = bestDetection.detection.score;

    // Calculate position metrics
    const position = calculatePosition(box, video.videoWidth, video.videoHeight);

    // Extract landmarks
    const extractedLandmarks = extractLandmarks(landmarks);

    // Calculate quality
    const quality = calculateQuality(score, position, extractedLandmarks);

    return {
      detected: true,
      faceCount: detections.length,
      position,
      landmarks: extractedLandmarks,
      quality,
      rawScore: score,
      timestamp,
    };
  } catch (error) {
    console.error('Face detection error:', error);
    return {
      detected: false,
      faceCount: 0,
      position: null,
      landmarks: null,
      quality: null,
      rawScore: 0,
      timestamp,
    };
  }
}

/**
 * Calculate face position metrics
 */
function calculatePosition(
  box: faceapi.Box,
  imageWidth: number,
  imageHeight: number
): FacePosition {
  const config = DEFAULT_DETECTION_CONFIG;

  // Calculate face size as percentage of image
  const faceArea = box.width * box.height;
  const imageArea = imageWidth * imageHeight;
  const sizeRatio = faceArea / imageArea;

  // Calculate center offset
  const faceCenterX = box.x + box.width / 2;
  const faceCenterY = box.y + box.height / 2;
  const imageCenterX = imageWidth / 2;
  const imageCenterY = imageHeight / 2;

  const offsetX = Math.abs(faceCenterX - imageCenterX) / imageWidth;
  const offsetY = Math.abs(faceCenterY - imageCenterY) / imageHeight;
  const centerOffset = Math.sqrt(offsetX * offsetX + offsetY * offsetY);

  // Validate against config
  const isSizeValid = sizeRatio >= config.minFaceSize && sizeRatio <= config.maxFaceSize;
  const isCentered = centerOffset <= config.maxCenterOffset;

  return {
    boundingBox: {
      x: box.x,
      y: box.y,
      width: box.width,
      height: box.height,
    },
    center: {
      x: faceCenterX,
      y: faceCenterY,
    },
    sizeRatio,
    centerOffset,
    isSizeValid,
    isCentered,
  };
}

/**
 * Extract key landmarks from face-api detection
 */
function extractLandmarks(landmarks: faceapi.FaceLandmarks68): FaceLandmarks {
  const positions = landmarks.positions;

  // face-api.js 68 landmark positions:
  // 0-16: jaw
  // 17-21: left eyebrow
  // 22-26: right eyebrow
  // 27-35: nose
  // 36-41: left eye
  // 42-47: right eye
  // 48-67: mouth

  const toPoint = (p: faceapi.Point): Point => ({ x: p.x, y: p.y });

  // Left eye center (average of points 36-41)
  const leftEyePoints = positions.slice(36, 42);
  const leftEye = {
    x: leftEyePoints.reduce((sum, p) => sum + p.x, 0) / leftEyePoints.length,
    y: leftEyePoints.reduce((sum, p) => sum + p.y, 0) / leftEyePoints.length,
  };

  // Right eye center (average of points 42-47)
  const rightEyePoints = positions.slice(42, 48);
  const rightEye = {
    x: rightEyePoints.reduce((sum, p) => sum + p.x, 0) / rightEyePoints.length,
    y: rightEyePoints.reduce((sum, p) => sum + p.y, 0) / rightEyePoints.length,
  };

  // Nose tip (point 30)
  const nosePoint = positions[30];
  const nose = nosePoint ? toPoint(nosePoint) : { x: 0, y: 0 };

  // Mouth center (average of points 48-67)
  const mouthPoints = positions.slice(48, 68);
  const mouth = {
    x: mouthPoints.reduce((sum, p) => sum + p.x, 0) / mouthPoints.length,
    y: mouthPoints.reduce((sum, p) => sum + p.y, 0) / mouthPoints.length,
  };

  // Mouth corners
  const leftMouthPoint = positions[48];
  const rightMouthPoint = positions[54];
  const leftMouth = leftMouthPoint ? toPoint(leftMouthPoint) : { x: 0, y: 0 };
  const rightMouth = rightMouthPoint ? toPoint(rightMouthPoint) : { x: 0, y: 0 };

  // Jaw outline (points 0-16)
  const jawOutline = positions.slice(0, 17).map(toPoint);

  return {
    leftEye,
    rightEye,
    nose,
    mouth,
    leftMouth,
    rightMouth,
    jawOutline,
    raw: positions.map(toPoint),
  };
}

/**
 * Calculate detection quality metrics
 */
function calculateQuality(
  score: number,
  position: FacePosition,
  landmarks: FaceLandmarks
): DetectionQuality {
  const config = DEFAULT_DETECTION_CONFIG;

  // Base quality from detection score
  const baseScore = score;

  // Estimate lighting from face size consistency
  // (poor lighting often causes inconsistent detection)
  const lightingScore = Math.min(1, score * 1.2);

  // Estimate sharpness from landmark consistency
  // (blurry images have less precise landmarks)
  const eyeDistance = Math.sqrt(
    Math.pow(landmarks.rightEye.x - landmarks.leftEye.x, 2) +
    Math.pow(landmarks.rightEye.y - landmarks.leftEye.y, 2)
  );
  const expectedEyeDistance = position.boundingBox.width * 0.4;
  const eyeRatio = eyeDistance / expectedEyeDistance;
  const sharpnessScore = Math.min(1, Math.max(0, 1 - Math.abs(1 - eyeRatio)));

  // Check if eyes are visible (not too close together = closed)
  const eyesVisible = eyeDistance > position.boundingBox.width * 0.2;

  // Check if mouth is visible (has reasonable width)
  const mouthWidth = Math.abs(landmarks.rightMouth.x - landmarks.leftMouth.x);
  const mouthVisible = mouthWidth > position.boundingBox.width * 0.15;

  // Overall quality
  const overallScore = (baseScore + lightingScore + sharpnessScore) / 3;
  const isAcceptable = overallScore >= config.minDetectionScore &&
    position.isSizeValid &&
    position.isCentered &&
    eyesVisible;

  return {
    score: overallScore,
    lightingScore,
    sharpnessScore,
    eyesVisible,
    mouthVisible,
    isAcceptable,
  };
}

/**
 * Reset service state (for testing or cleanup)
 */
export function resetService(): void {
  modelsLoaded = false;
  loadingPromise = null;
  loadedModels.clear();
  loadingState = {
    isLoading: false,
    isLoaded: false,
    error: null,
    progress: 0,
    loadedModels: [],
  };
}
