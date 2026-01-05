import type { RekkoError, RekkoErrorCode } from '@/types';

export function createError(code: RekkoErrorCode, message: string, details?: unknown): RekkoError {
  return { code, message, details };
}

// Mensagens amigáveis em português para cada código de erro
const friendlyMessages = {
  CAMERA_DENIED: 'Você negou o acesso à câmera. Permita o acesso para continuar.',
  CAMERA_NOT_FOUND: 'Nenhuma câmera encontrada no seu dispositivo.',
  CAMERA_ERROR: 'Erro ao acessar a câmera. Tente novamente.',
  CAMERA_IN_USE: 'A câmera está sendo usada por outro aplicativo.',
  DOMAIN_NOT_ALLOWED: 'Este site não está autorizado a usar o widget.',
  ORIGIN_NOT_ALLOWED: 'Este site não está autorizado a usar o widget.',
  INVALID_PUBLIC_KEY: 'Chave de acesso inválida. Contate o suporte.',
  SESSION_EXPIRED: 'Sua sessão expirou. Recarregue a página e tente novamente.',
  SESSION_NOT_FOUND: 'Sessão não encontrada. Recarregue a página.',
  NETWORK_ERROR: 'Erro de conexão. Verifique sua internet e tente novamente.',
  VERIFICATION_FAILED: 'Não foi possível verificar sua identidade. Tente novamente.',
  REGISTRATION_FAILED: 'Não foi possível cadastrar seu rosto. Tente novamente.',
  FACE_NOT_FOUND: 'Nenhum rosto cadastrado para este usuário. Faça o cadastro primeiro.',
  FACE_ALREADY_EXISTS: 'Este usuário já possui um rosto cadastrado.',
  NO_FACE_DETECTED: 'Nenhum rosto detectado. Posicione-se melhor na câmera.',
  MULTIPLE_FACES: 'Múltiplos rostos detectados. Apenas uma pessoa deve aparecer.',
  LOW_QUALITY: 'Qualidade da imagem muito baixa. Melhore a iluminação.',
  LIVENESS_FAILED: 'Verificação de vivacidade falhou. Certifique-se de estar ao vivo.',
  UNKNOWN_ERROR: 'Ocorreu um erro inesperado. Tente novamente.',
} as const;

type FriendlyMessageKey = keyof typeof friendlyMessages;

export const Errors = {
  cameraDenied: (): RekkoError =>
    createError('CAMERA_DENIED', friendlyMessages.CAMERA_DENIED),

  cameraNotFound: (): RekkoError =>
    createError('CAMERA_NOT_FOUND', friendlyMessages.CAMERA_NOT_FOUND),

  cameraError: (details?: unknown): RekkoError =>
    createError('CAMERA_ERROR', friendlyMessages.CAMERA_ERROR, details),

  cameraInUse: (): RekkoError =>
    createError('CAMERA_IN_USE', friendlyMessages.CAMERA_IN_USE),

  domainNotAllowed: (): RekkoError =>
    createError('DOMAIN_NOT_ALLOWED', friendlyMessages.DOMAIN_NOT_ALLOWED),

  originNotAllowed: (): RekkoError =>
    createError('ORIGIN_NOT_ALLOWED', friendlyMessages.ORIGIN_NOT_ALLOWED),

  invalidPublicKey: (): RekkoError =>
    createError('INVALID_PUBLIC_KEY', friendlyMessages.INVALID_PUBLIC_KEY),

  sessionExpired: (): RekkoError =>
    createError('SESSION_EXPIRED', friendlyMessages.SESSION_EXPIRED),

  sessionNotFound: (): RekkoError =>
    createError('SESSION_NOT_FOUND', friendlyMessages.SESSION_NOT_FOUND),

  networkError: (details?: unknown): RekkoError =>
    createError('NETWORK_ERROR', friendlyMessages.NETWORK_ERROR, details),

  verificationFailed: (details?: unknown): RekkoError =>
    createError('VERIFICATION_FAILED', friendlyMessages.VERIFICATION_FAILED, details),

  registrationFailed: (details?: unknown): RekkoError =>
    createError('REGISTRATION_FAILED', friendlyMessages.REGISTRATION_FAILED, details),

  faceNotFound: (): RekkoError =>
    createError('FACE_NOT_FOUND', friendlyMessages.FACE_NOT_FOUND),

  faceAlreadyExists: (): RekkoError =>
    createError('FACE_ALREADY_EXISTS', friendlyMessages.FACE_ALREADY_EXISTS),

  noFaceDetected: (): RekkoError =>
    createError('NO_FACE_DETECTED', friendlyMessages.NO_FACE_DETECTED),

  multipleFaces: (): RekkoError =>
    createError('MULTIPLE_FACES', friendlyMessages.MULTIPLE_FACES),

  lowQuality: (): RekkoError =>
    createError('LOW_QUALITY', friendlyMessages.LOW_QUALITY),

  livenessFailed: (): RekkoError =>
    createError('LIVENESS_FAILED', friendlyMessages.LIVENESS_FAILED),

  unknown: (details?: unknown): RekkoError =>
    createError('UNKNOWN_ERROR', friendlyMessages.UNKNOWN_ERROR, details),
};

// Mapeia códigos de erro da API para erros do widget
export function fromApiError(apiCode: string, apiMessage?: string): RekkoError {
  const isKnownCode = apiCode in friendlyMessages;
  if (isKnownCode) {
    const code = apiCode as FriendlyMessageKey;
    return createError(code as RekkoErrorCode, friendlyMessages[code]);
  }
  return createError('UNKNOWN_ERROR', apiMessage || friendlyMessages.UNKNOWN_ERROR);
}

export function isRekkoError(error: unknown): error is RekkoError {
  return (
    typeof error === 'object' &&
    error !== null &&
    'code' in error &&
    'message' in error
  );
}
