export { ApiClient } from './api';
export type {
  SessionResponse,
  VerifyResponse,
  RegisterResponse,
  LivenessResponse,
} from './api';

// Face detection service
export {
  loadModels,
  detectFaces,
  areModelsLoaded,
  getLoadingState,
  resetService,
} from './faceDetection';
