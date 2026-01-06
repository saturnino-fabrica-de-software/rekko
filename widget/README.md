# Rekko Widget

Widget embarcável de reconhecimento facial para integração white-label em aplicações web.

[![TypeScript](https://img.shields.io/badge/TypeScript-5.6-blue.svg)](https://www.typescriptlang.org/)
[![Preact](https://img.shields.io/badge/Preact-10.24-purple.svg)](https://preactjs.com/)
[![Vite](https://img.shields.io/badge/Vite-5.4-yellow.svg)](https://vitejs.dev/)
[![License](https://img.shields.io/badge/License-Proprietary-red.svg)]()

## Visao Geral

O Rekko Widget oferece uma experiencia completa de reconhecimento facial com:

- **Deteccao facial em tempo real** via face-api.js
- **Liveness detection** (prova de vida) com desafios ativos
- **Fluxo guiado** do consentimento ate o resultado
- **Customizacao total** de tema e textos (i18n)
- **LGPD compliant** com tela de consentimento obrigatoria

## Stack Tecnologica

| Tecnologia | Versao | Uso |
|------------|--------|-----|
| **Preact** | 10.24 | UI framework (3KB vs React 40KB) |
| **TypeScript** | 5.6 | Type safety |
| **Vite** | 5.4 | Build tool |
| **face-api.js** | @vladmandic/face-api | Deteccao facial client-side |
| **CSS Modules** | - | Estilos encapsulados |

## Arquitetura

```
┌─────────────────────────────────────────────────────────────────┐
│                         Rekko Widget                            │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │   Rekko     │  │   Widget    │  │   Modal     │              │
│  │  (Facade)   │──│ (Container) │──│  (Layout)   │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
│         │                                                        │
│         ▼                                                        │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                        Screens                               ││
│  │  ┌─────────┐ ┌───────────┐ ┌────────┐ ┌──────────┐          ││
│  │  │ Consent │→│Orientation│→│ Camera │→│ Liveness │          ││
│  │  └─────────┘ └───────────┘ └────────┘ └──────────┘          ││
│  │                                │              │              ││
│  │                                ▼              ▼              ││
│  │                         ┌────────────┐ ┌──────────┐          ││
│  │                         │ Processing │→│  Result  │          ││
│  │                         └────────────┘ └──────────┘          ││
│  └─────────────────────────────────────────────────────────────┘│
│         │                                                        │
│         ▼                                                        │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                         Hooks                                ││
│  │  ┌───────────────┐  ┌─────────────┐  ┌──────────────┐       ││
│  │  │useFaceDetection│  │ useLiveness │  │  useCamera   │       ││
│  │  └───────────────┘  └─────────────┘  └──────────────┘       ││
│  └─────────────────────────────────────────────────────────────┘│
│         │                                                        │
│         ▼                                                        │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                       Services                               ││
│  │  ┌───────────────┐  ┌─────────────┐                         ││
│  │  │ faceDetection │  │  ApiClient  │                         ││
│  │  │  (face-api)   │  │   (HTTP)    │                         ││
│  │  └───────────────┘  └─────────────┘                         ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

## Estrutura de Pastas

```
widget/
├── src/
│   ├── components/           # Componentes UI
│   │   ├── CameraScreen/     # Tela de camera + deteccao
│   │   │   ├── CameraScreen.tsx
│   │   │   ├── CameraScreen.module.css
│   │   │   ├── FaceDetector.tsx      # Overlay de deteccao
│   │   │   └── FaceDetector.module.css
│   │   ├── ConsentScreen/    # Tela LGPD
│   │   ├── OrientationScreen/# Instrucoes pre-camera
│   │   ├── LivenessScreen/   # Prova de vida ativa
│   │   ├── ProcessingScreen/ # Loading durante API call
│   │   ├── ResultScreen/     # Sucesso/Erro
│   │   ├── Modal/            # Container modal
│   │   └── Widget.tsx        # Orquestrador de telas
│   │
│   ├── hooks/                # React hooks
│   │   ├── useCamera.ts      # Gerenciamento de camera
│   │   ├── useFaceDetection.ts # Deteccao facial
│   │   └── useLiveness.ts    # Desafios de liveness
│   │
│   ├── services/             # Servicos externos
│   │   ├── api.ts            # Cliente HTTP
│   │   └── faceDetection.ts  # Wrapper face-api.js
│   │
│   ├── config/               # Configuracoes
│   │   └── detection.ts      # Thresholds de deteccao
│   │
│   ├── locales/              # Textos i18n
│   │   ├── pt-BR.ts          # Portugues default
│   │   └── index.ts
│   │
│   ├── styles/               # Estilos globais
│   │   ├── variables.css     # CSS variables
│   │   └── theme.ts          # Aplicacao de tema
│   │
│   ├── types/                # TypeScript types
│   │   ├── index.ts          # Tipos principais
│   │   └── faceDetection.ts  # Tipos de deteccao
│   │
│   ├── errors/               # Tratamento de erros
│   │   └── index.ts          # Error factory
│   │
│   ├── Rekko.tsx             # Classe principal (facade)
│   └── index.ts              # Entry point
│
├── dist/                     # Build de producao
├── docs/                     # Documentacao adicional
│   └── INTEGRATION.md        # Guia de integracao
├── index.html                # Dev server page
├── vite.config.ts            # Config Vite
├── tsconfig.json             # Config TypeScript
└── package.json
```

## Fluxo de Estados

```
                    ┌─────────┐
                    │  idle   │ (widget fechado)
                    └────┬────┘
                         │ open()
                         ▼
                    ┌─────────┐
                    │ consent │ (aceitar LGPD)
                    └────┬────┘
                         │ aceitar
                         ▼
                  ┌─────────────┐
                  │ orientation │ (instrucoes)
                  └──────┬──────┘
                         │ continuar
                         ▼
                    ┌─────────┐
                    │ camera  │ (captura foto)
                    └────┬────┘
                         │
         ┌───────────────┴───────────────┐
         │                               │
    mode=register                   mode=verify
         │                               │
         ▼                               │
    ┌──────────┐                         │
    │ liveness │ (prova de vida)         │
    └────┬─────┘                         │
         │                               │
         └───────────────┬───────────────┘
                         │
                         ▼
                  ┌────────────┐
                  │ processing │ (API call)
                  └─────┬──────┘
                        │
                        ▼
                   ┌─────────┐
                   │ result  │ (sucesso/erro)
                   └─────────┘
```

## Deteccao Facial

### face-api.js

O widget usa [@vladmandic/face-api](https://github.com/vladmandic/face-api) para deteccao facial client-side.

**Modelos carregados (lazy loading via CDN):**
- `tiny_face_detector` - Deteccao de faces (190KB)
- `face_landmark_68_tiny` - Pontos faciais (300KB)

**Thresholds de deteccao** (`config/detection.ts`):

```typescript
export const DETECTION_CONFIG = {
  // Tamanho minimo do rosto (% da tela)
  MIN_FACE_SIZE: 0.15,
  MAX_FACE_SIZE: 0.85,

  // Centralizacao
  CENTER_TOLERANCE: 0.15,

  // Qualidade
  MIN_CONFIDENCE: 0.5,

  // Timing
  COUNTDOWN_SECONDS: 3,
  DETECTION_INTERVAL_MS: 150,
};
```

### Estados de Deteccao

| Estado | Cor | Descricao |
|--------|-----|-----------|
| `initializing` | Cinza | Carregando modelos |
| `no_face` | Vermelho | Nenhum rosto detectado |
| `face_too_small` | Laranja | Aproxime-se da camera |
| `face_too_large` | Laranja | Afaste-se da camera |
| `face_not_centered` | Amarelo | Centralize o rosto |
| `multiple_faces` | Vermelho | Apenas uma pessoa |
| `poor_lighting` | Amarelo | Melhore a iluminacao |
| `ready` | Verde | Pronto para captura |
| `countdown` | Verde pulsando | Contagem regressiva |

## Liveness Detection

### Desafios Ativos

O widget implementa **Active Liveness Detection** com desafios que o usuario deve completar:

| Desafio | Deteccao | Threshold |
|---------|----------|-----------|
| `turn_right` | Calculo de yaw via landmarks | 20 graus |
| `turn_left` | Calculo de yaw via landmarks | 20 graus |
| `blink` | Mudanca de visibilidade dos olhos | 3 frames |

### Calculo de Yaw (Rotacao da Cabeca)

```typescript
const calculateYaw = (landmarks) => {
  const eyeCenter = (landmarks.leftEye.x + landmarks.rightEye.x) / 2;
  const eyeDistance = Math.abs(landmarks.rightEye.x - landmarks.leftEye.x);
  const noseOffset = landmarks.nose.x - eyeCenter;
  return (noseOffset / eyeDistance) * 60; // graus
};
```

### Configuracao de Liveness

```typescript
const LIVENESS_CONFIG = {
  challenges: ['turn_right', 'blink'], // Desafios a executar
  timeoutMs: 10000,                    // Timeout por desafio
  maxAttempts: 3,                      // Tentativas maximas
  turnThreshold: 20,                   // Graus para virar
  blinkFrames: 3,                      // Frames para piscar
};
```

## API Reference

### Rekko.init(config)

Inicializa o widget com configuracoes globais.

```typescript
interface RekkoConfig {
  publicKey: string;              // Chave publica do tenant
  locale: string;                 // Locale (ex: 'pt-BR')
  texts: Record<string, LocaleTexts>; // Textos por locale
  theme?: RekkoTheme;             // Tema customizado
  logo?: string;                  // URL do logo
  apiUrl?: string;                // URL da API (default: https://api.rekko.io)
}
```

### Rekko.open(options)

Abre o widget.

```typescript
interface RekkoOpenOptions {
  mode: 'register' | 'verify';    // Modo de operacao
  externalId?: string;            // ID do usuario no sistema cliente
  onSuccess: (result: RekkoResult) => void;
  onError: (error: RekkoError) => void;
  onEvent?: (event: RekkoEvent) => void;
}
```

### Rekko.close()

Fecha o widget.

### Rekko.isInitialized()

Retorna `true` se o widget foi inicializado.

## Eventos

```typescript
type RekkoEventType =
  | 'widget_opened'        // Widget aberto
  | 'widget_closed'        // Widget fechado
  | 'consent_accepted'     // Consentimento aceito
  | 'consent_declined'     // Consentimento recusado
  | 'camera_ready'         // Camera pronta
  | 'camera_error'         // Erro de camera
  | 'face_detected'        // Rosto detectado
  | 'face_lost'            // Rosto perdido
  | 'face_detection_state' // Mudanca de estado de deteccao
  | 'liveness_started'     // Liveness iniciado
  | 'liveness_challenge'   // Desafio de liveness
  | 'liveness_success'     // Liveness sucesso
  | 'liveness_failed'      // Liveness falhou
  | 'capture_started'      // Captura iniciada
  | 'processing'           // Processando
  | 'verification_success' // Verificacao OK
  | 'verification_failed'  // Verificacao falhou
  | 'registration_success' // Registro OK
  | 'registration_failed'; // Registro falhou
```

## Customizacao de Tema

```typescript
interface RekkoTheme {
  primaryColor?: string;        // Cor primaria (botoes, bordas ativas)
  primaryHoverColor?: string;   // Cor hover
  backgroundColor?: string;     // Fundo do modal
  surfaceColor?: string;        // Fundo de cards
  textColor?: string;           // Texto principal
  textSecondaryColor?: string;  // Texto secundario
  borderColor?: string;         // Bordas
  borderRadius?: string;        // Arredondamento
  fontFamily?: string;          // Fonte
}
```

## Internacionalizacao (i18n)

### Textos Default (pt-BR)

```typescript
import { Rekko, ptBR } from '@rekko/widget';

Rekko.init({
  publicKey: 'pk_...',
  locale: 'pt-BR',
  texts: {
    'pt-BR': ptBR,
  },
});
```

### Estrutura de Textos

```typescript
interface LocaleTexts {
  consent: { title, body, accept, decline };
  orientation: { title, subtitle, instructions: {...}, continue };
  camera: { title, instruction, positioning, capturing };
  liveness: { title, challenges: {...}, success, failed, timeout, retry, skip };
  processing: { title, message };
  result: { successTitle, successMessage, errorTitle, errorMessage, retry, close };
  errors: { cameraPermission, cameraNotFound, cameraError, networkError, ... };
}
```

## Desenvolvimento

### Pre-requisitos

- Node.js 18+
- npm 9+

### Setup

```bash
cd widget
npm install
```

### Dev Server

```bash
npm run dev
# Abre http://localhost:5173
```

### Build

```bash
npm run build
```

**Output:**
- `dist/rekko.min.js` - Bundle UMD (1.3MB, 356KB gzip)
- `dist/rekko.css` - Estilos (18KB, 4KB gzip)
- `dist/rekko.es.js` - ESM entry

### Type Check

```bash
npm run typecheck
```

### Testes

```bash
npm run test
```

## Performance

### Bundle Size

| Arquivo | Tamanho | Gzip |
|---------|---------|------|
| `rekko.min.js` | 1.37 MB | 356 KB |
| `rekko.css` | 18 KB | 4 KB |

> O bundle e grande devido ao face-api.js (~1MB). Os modelos de ML sao carregados sob demanda via CDN.

### Lazy Loading

- **Widget**: Code-split via `import()` dinamico
- **Modelos ML**: Carregados apenas quando camera e ativada
- **CDN**: Modelos servidos via jsDelivr (cache global)

## Seguranca

### LGPD Compliance

- Tela de consentimento obrigatoria antes de acessar camera
- Texto configuravel pelo integrador
- Evento `consent_accepted/declined` para auditoria

### Protecao de Dominio

- Validacao de origem no backend
- Public key vinculada a dominios permitidos
- CORS configurado por tenant

### Dados Biometricos

- Imagens processadas e descartadas apos uso
- Embeddings criptografados at-rest (backend)
- Nenhum dado biometrico armazenado no browser

## Compatibilidade

### Browsers

| Browser | Versao | Suporte |
|---------|--------|---------|
| Chrome | 80+ | Full |
| Firefox | 75+ | Full |
| Safari | 14+ | Full |
| Edge | 80+ | Full |

### Requisitos

- HTTPS obrigatorio (exceto localhost)
- Camera frontal
- WebGL support (para face-api.js)

## Troubleshooting

### Modelos nao carregam

```
Error: Failed to load face detection models
```

**Causa**: CDN bloqueado ou sem conexao
**Solucao**: Widget automaticamente mostra botao de captura manual

### Camera nao funciona

```
Error: CAMERA_DENIED
```

**Causa**: Permissao negada pelo usuario
**Solucao**: Mostrar instrucoes para habilitar camera nas configuracoes do browser

### Liveness falha repetidamente

**Causa**: Iluminacao ruim ou movimentos muito rapidos
**Solucao**:
- Melhorar iluminacao
- Fazer movimentos mais lentos
- Usar botao "Pular" se disponivel

## Changelog

### v0.1.0

- Setup inicial com Preact + Vite
- Telas: Consent, Camera, Processing, Result
- Deteccao facial via face-api.js
- Liveness detection com desafios ativos
- OrientationScreen com instrucoes
- Tema customizavel
- i18n com textos em pt-BR
- Build otimizado com code-splitting

## License

Proprietary - Rekko Technologies
