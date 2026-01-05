import type { LocaleTexts } from '@/types';

export const ptBR: LocaleTexts = {
  consent: {
    title: 'Verificação Facial',
    body: 'Para continuar, precisamos capturar sua face. Seus dados biométricos serão processados de acordo com a LGPD.',
    accept: 'Aceitar e Continuar',
    decline: 'Recusar',
  },
  orientation: {
    title: 'Prepare-se para a Foto',
    subtitle: 'Siga as instruções para uma captura de qualidade',
    instructions: {
      neutral: 'Mantenha expressão neutra',
      visible: 'Rosto totalmente visível',
      lighting: 'Boa iluminação no rosto',
      framing: 'Centralize o rosto na tela',
    },
    continue: 'Estou Pronto',
  },
  camera: {
    title: 'Posicione seu Rosto',
    instruction: 'Centralize seu rosto no círculo',
    positioning: 'Ajuste sua posição...',
    capturing: 'Capturando...',
  },
  liveness: {
    title: 'Verificação de Identidade',
    challenges: {
      turn_left: 'Vire a cabeça para a esquerda',
      turn_right: 'Vire a cabeça para a direita',
      blink: 'Pisque os olhos',
    },
    success: 'Verificação concluída!',
    failed: 'Verificação de vivacidade falhou. Tente novamente.',
    timeout: 'Tempo esgotado. Tente novamente.',
    retry: 'Tentar Novamente',
    skip: 'Pular esta etapa',
  },
  processing: {
    title: 'Processando',
    message: 'Aguarde enquanto verificamos sua identidade...',
  },
  result: {
    successTitle: 'Verificado!',
    successMessage: 'Sua identidade foi confirmada com sucesso.',
    errorTitle: 'Falha na Verificação',
    errorMessage: 'Não foi possível verificar sua identidade.',
    retry: 'Tentar Novamente',
    close: 'Fechar',
  },
  errors: {
    cameraPermission: 'Permissão da câmera negada',
    cameraNotFound: 'Nenhuma câmera encontrada',
    cameraError: 'Erro ao acessar a câmera',
    networkError: 'Erro de conexão',
    verificationFailed: 'Falha na verificação',
    registrationFailed: 'Falha no registro',
  },
};

export default ptBR;
