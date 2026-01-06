# Rekko Widget - Guia de Integração

Este documento descreve como integrar o widget de reconhecimento facial Rekko em seu site.

## Requisitos

1. **Public Key**: Obtenha sua `public_key` no painel administrativo do Rekko
2. **Domínios Permitidos**: Configure os domínios autorizados a usar o widget

## Instalação

### Via CDN (Recomendado)

```html
<!-- CSS do Widget -->
<link rel="stylesheet" href="https://cdn.rekko.io/widget/rekko.css">

<!-- JS do Widget -->
<script src="https://cdn.rekko.io/widget/rekko.min.js"></script>
```

### Via NPM

```bash
npm install @rekko/widget
```

```javascript
import { Rekko } from '@rekko/widget';
import '@rekko/widget/dist/rekko.css';
```

## Configuração

### 1. Inicialização

```javascript
Rekko.init({
  publicKey: 'pk_live_sua_chave_aqui',
  apiUrl: 'https://api.rekko.io', // Opcional - URL da API
  locale: 'pt-BR',
  texts: {
    'pt-BR': {
      consent: {
        title: 'Verificação Facial',
        body: 'Para continuar, precisamos capturar sua imagem facial. Seus dados biométricos serão processados de acordo com a LGPD.',
        accept: 'Aceitar e Continuar',
        decline: 'Cancelar'
      },
      camera: {
        title: 'Posicione seu Rosto',
        instruction: 'Centralize seu rosto no círculo',
        positioning: 'Ajustando posição...',
        capturing: 'Capturando...'
      },
      processing: {
        title: 'Processando',
        message: 'Analisando sua imagem facial...'
      },
      result: {
        successTitle: 'Sucesso!',
        successMessage: 'Verificação concluída com sucesso.',
        errorTitle: 'Erro',
        errorMessage: 'Não foi possível completar a verificação.',
        retry: 'Tentar Novamente',
        close: 'Fechar'
      },
      errors: {
        cameraPermission: 'Permissão de câmera negada',
        cameraNotFound: 'Câmera não encontrada',
        cameraError: 'Erro ao acessar câmera',
        networkError: 'Erro de conexão',
        verificationFailed: 'Falha na verificação',
        registrationFailed: 'Falha no cadastro'
      }
    }
  },
  // Tema customizado (opcional)
  theme: {
    primaryColor: '#00d9ff',
    primaryHoverColor: '#00b8d9',
    backgroundColor: '#ffffff',
    surfaceColor: '#f5f5f5',
    textColor: '#1a1a1a',
    textSecondaryColor: '#666666',
    borderColor: '#e0e0e0',
    borderRadius: '12px',
    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif'
  },
  // Logo customizado (opcional)
  logo: 'https://seu-site.com/logo.png'
});
```

### 2. Abrir Widget

#### Modo Verificação (1:1)

```javascript
Rekko.open({
  mode: 'verify',
  externalId: 'user-123', // ID do usuário no seu sistema
  onSuccess: (result) => {
    console.log('Verificação:', result.verified);
    console.log('Confiança:', result.confidence);
    // result = { mode: 'verify', verified: true, confidence: 0.95, faceId: '...', externalId: 'user-123' }
  },
  onError: (error) => {
    console.error('Erro:', error.code, error.message);
  },
  onEvent: (event) => {
    console.log('Evento:', event.type, event.data);
  }
});
```

#### Modo Cadastro

```javascript
Rekko.open({
  mode: 'register',
  externalId: 'user-123', // ID do usuário no seu sistema
  onSuccess: (result) => {
    console.log('Face cadastrada:', result.faceId);
    // result = { mode: 'register', registered: true, faceId: '...', externalId: 'user-123' }
  },
  onError: (error) => {
    console.error('Erro:', error.code, error.message);
  },
  onEvent: (event) => {
    console.log('Evento:', event.type);
  }
});
```

### 3. Fechar Widget

```javascript
Rekko.close();
```

## Eventos

O callback `onEvent` recebe os seguintes eventos:

| Evento | Descrição |
|--------|-----------|
| `widget_opened` | Widget foi aberto |
| `widget_closed` | Widget foi fechado |
| `consent_accepted` | Usuário aceitou os termos |
| `consent_declined` | Usuário recusou os termos |
| `camera_ready` | Câmera está pronta |
| `camera_error` | Erro ao acessar câmera |
| `face_detected` | Rosto detectado (início do countdown) |
| `capture_started` | Captura iniciada |
| `processing` | Processando imagem |
| `verification_success` | Verificação bem-sucedida |
| `verification_failed` | Verificação falhou |
| `registration_success` | Cadastro bem-sucedido |
| `registration_failed` | Cadastro falhou |

## Códigos de Erro

| Código | Descrição | Solução |
|--------|-----------|---------|
| `CAMERA_DENIED` | Permissão de câmera negada | Solicite permissão ao usuário |
| `CAMERA_NOT_FOUND` | Câmera não encontrada | Verifique se o dispositivo tem câmera |
| `CAMERA_IN_USE` | Câmera em uso por outro app | Feche outros apps que usam câmera |
| `DOMAIN_NOT_ALLOWED` | Domínio não autorizado | Adicione o domínio no painel |
| `INVALID_PUBLIC_KEY` | Public key inválida | Verifique a public_key |
| `SESSION_EXPIRED` | Sessão expirada | Reabra o widget |
| `NETWORK_ERROR` | Erro de conexão | Verifique a conexão com internet |
| `NO_FACE_DETECTED` | Nenhum rosto detectado | Posicione o rosto corretamente |
| `MULTIPLE_FACES` | Múltiplos rostos | Apenas uma pessoa na câmera |
| `LOW_QUALITY` | Qualidade baixa | Melhore iluminação |
| `FACE_NOT_FOUND` | Face não cadastrada | Cadastre antes de verificar |
| `LIVENESS_FAILED` | Falha no liveness | Tente novamente |

## Configuração do Tenant

### Domínios Permitidos

Configure os domínios que podem usar o widget no painel administrativo.

**Importante**: Inclua a porta se estiver usando uma diferente da padrão.

Exemplos:
- `meusite.com.br` (produção)
- `localhost:3000` (desenvolvimento)
- `localhost:5173` (Vite dev server)
- `localhost:5500` (Live Server)

### Public Key

A public key tem o formato: `pk_<env>_<32 caracteres>`

- `pk_live_*` - Produção
- `pk_test_*` - Testes/Sandbox

## Troubleshooting

### Erro: "Origin domain is not allowed"

**Causa**: O domínio atual não está na lista de domínios permitidos.

**Solução**:
1. Acesse o painel administrativo
2. Vá em Configurações > Widget
3. Adicione o domínio (incluindo porta se aplicável)

### Erro: "Invalid or inactive public key"

**Causa**: A public_key está incorreta ou o tenant está inativo.

**Solução**:
1. Verifique se a public_key está correta
2. Verifique se o tenant está ativo no painel

### Widget não abre

**Causa**: Widget não foi inicializado corretamente.

**Solução**:
```javascript
// Verifique se está inicializado
console.log('Inicializado:', Rekko.isInitialized());

// Inicialize antes de usar
Rekko.init({ ... });
```

### Câmera não funciona

**Causas possíveis**:
1. Permissão negada
2. Página não está em HTTPS (exceto localhost)
3. Câmera em uso por outro aplicativo

**Solução**:
1. Solicite permissão de câmera
2. Use HTTPS em produção
3. Feche outros apps que usam a câmera

## Exemplo Completo

```html
<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Meu Site - Login com Face</title>
  <link rel="stylesheet" href="https://cdn.rekko.io/widget/rekko.css">
</head>
<body>
  <button id="btn-login">Login com Face</button>

  <script src="https://cdn.rekko.io/widget/rekko.min.js"></script>
  <script>
    // Inicializar
    Rekko.init({
      publicKey: 'pk_live_sua_chave_aqui',
      locale: 'pt-BR',
      texts: {
        'pt-BR': {
          consent: {
            title: 'Login com Reconhecimento Facial',
            body: 'Vamos verificar sua identidade usando reconhecimento facial.',
            accept: 'Continuar',
            decline: 'Cancelar'
          },
          camera: {
            title: 'Olhe para a Câmera',
            instruction: 'Posicione seu rosto no centro',
            positioning: 'Ajustando...',
            capturing: 'Capturando...'
          },
          processing: {
            title: 'Verificando',
            message: 'Aguarde um momento...'
          },
          result: {
            successTitle: 'Bem-vindo!',
            successMessage: 'Login realizado com sucesso.',
            errorTitle: 'Falha no Login',
            errorMessage: 'Não foi possível verificar sua identidade.',
            retry: 'Tentar Novamente',
            close: 'Fechar'
          },
          errors: {
            cameraPermission: 'Permita o acesso à câmera',
            cameraNotFound: 'Câmera não encontrada',
            cameraError: 'Erro na câmera',
            networkError: 'Sem conexão',
            verificationFailed: 'Verificação falhou',
            registrationFailed: 'Cadastro falhou'
          }
        }
      }
    });

    // Botão de login
    document.getElementById('btn-login').addEventListener('click', () => {
      const userId = 'user-123'; // ID do usuário logado

      Rekko.open({
        mode: 'verify',
        externalId: userId,
        onSuccess: (result) => {
          if (result.verified) {
            // Redirecionar para área logada
            window.location.href = '/dashboard';
          }
        },
        onError: (error) => {
          alert('Erro: ' + error.message);
        }
      });
    });
  </script>
</body>
</html>
```

## Suporte

- Email: suporte@rekko.io
- Documentação: https://docs.rekko.io
- Status: https://status.rekko.io
