# Widget - Ajustes Realizados Durante Testes

Este documento registra os ajustes feitos durante os testes do widget para referência futura.

## Problemas Encontrados e Soluções

### 1. Widget travava em "Analisando sua imagem facial..."

**Problema**: Após capturar a imagem, o widget ficava eternamente na tela de processing.

**Causa**: O `Widget.tsx` tinha `// TODO: Send to API` - a chamada à API nunca foi implementada.

**Solução**: Implementar a lógica completa em `Widget.tsx`:
- Criar sessão via API ao montar o componente
- Enviar imagem capturada para `/v1/widget/verify` ou `/v1/widget/register`
- Tratar resposta e mostrar tela de resultado

**Arquivo**: `widget/src/components/Widget.tsx`

---

### 2. Captura não era automática

**Problema**: Usuário precisava clicar no botão para capturar a imagem.

**Causa**: O `CameraScreen.tsx` só tinha captura manual, sem auto-captura.

**Solução**: Adicionar auto-captura com countdown:
- Após câmera ficar pronta, aguardar 1 segundo
- Iniciar countdown visual 3-2-1
- Capturar automaticamente ao final

**Arquivo**: `widget/src/components/CameraScreen/CameraScreen.tsx`

**CSS adicionado**: Animações de countdown em `CameraScreen.module.css`

---

### 3. API esperava multipart/form-data, frontend enviava JSON

**Problema**: Requisições de verify/register falhavam.

**Causa**: O handler Go usa `c.FormValue()` e espera imagem como arquivo, mas o frontend enviava JSON.

**Solução**: Alterar `api.ts` para usar FormData:
```typescript
private createFormData(externalId: string, imageBase64: string): FormData {
  const formData = new FormData();
  formData.append('session_id', this.sessionId!);
  formData.append('external_id', externalId);

  // Converter base64 para blob
  const blob = base64ToBlob(imageBase64);
  formData.append('image', blob, 'capture.jpg');

  return formData;
}
```

**Arquivo**: `widget/src/services/api.ts`

---

### 4. Campos da API em snake_case

**Problema**: API retornava `session_id`, frontend esperava `sessionId`.

**Causa**: Go usa snake_case nos JSON tags, frontend usava camelCase.

**Solução**: Mapear campos na resposta:
```typescript
const response = await this.request<{ session_id: string; expires_at: string }>(...);
this.sessionId = response.session_id;
return { sessionId: response.session_id, expiresAt: response.expires_at };
```

**Arquivo**: `widget/src/services/api.ts`

---

### 5. Erro "Origin domain is not allowed"

**Problema**: Widget não conseguia criar sessão.

**Causa**: Domínio `localhost:5500` não estava na lista de domínios permitidos.

**Solução**: Adicionar todos os domínios de desenvolvimento:
```sql
UPDATE tenants SET allowed_domains = ARRAY[
  'localhost',
  'localhost:5173',  -- Vite dev
  'localhost:5500',  -- Live Server
  '127.0.0.1',
  '127.0.0.1:5500'
] WHERE id = '<tenant_id>';
```

**Importante para documentação**: Orientar clientes a incluir a porta no domínio.

---

## Arquivos Modificados

| Arquivo | Alteração |
|---------|-----------|
| `widget/src/components/Widget.tsx` | Implementar chamada à API |
| `widget/src/components/CameraScreen/CameraScreen.tsx` | Auto-captura com countdown |
| `widget/src/components/CameraScreen/CameraScreen.module.css` | Animações do countdown |
| `widget/src/services/api.ts` | FormData + snake_case mapping |

## Checklist de Configuração do Tenant

Para o widget funcionar, o tenant precisa ter:

- [ ] `public_key` configurada (formato: `pk_<env>_<32chars>`)
- [ ] `allowed_domains` com todos os domínios (incluindo porta)
- [ ] Tenant ativo (`is_active = true`)

## Testes Recomendados

1. **Criar sessão**: Verificar se retorna `session_id` e `expires_at`
2. **Domínio inválido**: Verificar se retorna `ORIGIN_NOT_ALLOWED`
3. **Public key inválida**: Verificar se retorna `INVALID_PUBLIC_KEY`
4. **Verificação com face cadastrada**: Deve retornar `verified: true`
5. **Verificação sem face**: Deve retornar `FACE_NOT_FOUND`
6. **Cadastro de face**: Deve retornar `registered: true` e `faceId`

---

## Bugs Corrigidos - Janeiro 2026

### 6. Countdown de captura ficava "preso" em valores intermediários

**Problema**: O countdown mostrava valores inconsistentes como "2" travado ou pulava números.

**Causa**: Race condition no `useFaceDetection.ts` - múltiplas callbacks (`onStateChange`, `onFaceReady`) eram chamadas simultaneamente, cada uma fazendo `setDetectionInfo()` e sobrescrevendo o countdown da outra.

**Solução**: Usar **refs para callbacks** que quebram o ciclo de dependência do useCallback:

```typescript
// ❌ ERRADO: Callbacks com dependências circulares
const onStateChange = useCallback(() => {
  setDetectionInfo(prev => ({ ...prev, stabilityTime: newTime }));
}, [/* deps */]);

const onFaceReady = useCallback(() => {
  setDetectionInfo(prev => ({ ...prev, countdown: newCountdown }));
}, [onStateChange]); // <- Dependência que causa stale closure

// ✅ CORRETO: Usar refs para callbacks
const onFaceReadyRef = useRef<() => void>(() => {});

// Atualizar ref sempre que a função mudar
onFaceReadyRef.current = () => {
  setDetectionInfo(prev => ({ ...prev, countdown: newCountdown }));
};

// Usar ref.current() ao invés de callback direta
onFaceReadyRef.current();
```

**Arquivos**:
- `widget/src/hooks/useFaceDetection.ts`
- `widget/src/components/CameraScreen/FaceDetector.tsx`

**Padrão a seguir**: Sempre que tiver callbacks que se referenciam mutuamente em hooks React/Preact, usar refs para quebrar o ciclo.

---

### 7. Segundo challenge de liveness (turn_left) não detectava

**Problema**: Após completar turn_right com sucesso, o turn_left não detectava mesmo o usuário virando a cabeça corretamente. Sempre dava timeout.

**Causa**: Quando turn_left iniciava, o usuário ainda estava com a cabeça virada para a direita (posição final do challenge anterior). O `baseYawRef` capturava essa posição virada como "neutra", tornando impossível atingir o threshold de turn_left.

**Exemplo numérico**:
- Usuário completa turn_right com yaw = -20° (cabeça virada direita)
- Turn_left inicia, `baseYawRef = -20°`
- Threshold para turn_left: `diff > +15°`
- Usuário vira para esquerda, yaw = +15°
- `diff = 15 - (-20) = 35°` → Deveria detectar!
- **MAS**: O timing fazia o baseYaw ser capturado no momento errado

**Solução**: Implementar **wait-for-neutral** entre challenges:

```typescript
const NEUTRAL_YAW_THRESHOLD = 8; // Graus

// Estado para tracking
const waitingForNeutralRef = useRef<boolean>(false);

// Ao iniciar challenge que não é o primeiro
const shouldWaitForNeutral = completed.length > 0;
waitingForNeutralRef.current = shouldWaitForNeutral;

// Na detecção, primeiro verificar se precisa voltar ao centro
if (waitingForNeutralRef.current) {
  if (Math.abs(yaw) < NEUTRAL_YAW_THRESHOLD) {
    // Usuário voltou para frente, agora pode começar
    waitingForNeutralRef.current = false;
    baseYawRef.current = yaw; // Captura yaw neutro correto
    setState(prev => ({ ...prev, waitingForNeutral: false }));
  }
  return; // Não detecta challenge ainda
}
```

**UI Feedback**: Mostrar instrução "Olhe para frente" enquanto espera:

```typescript
const getCurrentInstruction = () => {
  if (state.waitingForNeutral) return 'Olhe para frente';
  return CHALLENGE_TEXTS[state.currentChallenge].instruction;
};
```

**Arquivos**:
- `widget/src/hooks/useLiveness.ts` - Lógica de detecção
- `widget/src/components/LivenessScreen/LivenessScreen.tsx` - UI feedback

---

### 8. DeepFace retornando 400 "Face could not be detected"

**Problema**: API retornava 500 nos endpoints `/v1/widget/validate` e `/v1/widget/register`.

**Causa**: DeepFace tem `enforce_detection: true` por padrão, rejeitando imagens onde a detecção facial tem baixa confiança (ângulos, iluminação, etc).

**Solução**: Enviar `enforce_detection: false` nas requisições:

```go
// internal/provider/deepface/client.go
func (c *Client) Represent(ctx context.Context, imageBase64 string) (*RepresentResponse, error) {
    enforceDetection := false  // ← Crítico!
    req := RepresentRequest{
        Img:              imageBase64,
        Model:            c.config.Model,
        Detector:         c.config.Detector,
        EnforceDetection: &enforceDetection,
    }
    // ...
}
```

**Por que usar ponteiro `*bool`?**
Go's `omitempty` não funciona com `bool` regular (false é zero-value e seria omitido).
Usando `*bool`, podemos enviar `false` explicitamente:

```go
type RepresentRequest struct {
    EnforceDetection *bool `json:"enforce_detection,omitempty"`
}
```

**Arquivo**: `internal/provider/deepface/client.go`, `internal/provider/deepface/models.go`

---

### 9. JSON field names incorretos para DeepFace API

**Problema**: DeepFace retornava erros ou ignorava parâmetros.

**Causa**: Os nomes dos campos JSON estavam errados:

```go
// ❌ ERRADO
Model    string `json:"model"`
Detector string `json:"detector"`

// ✅ CORRETO (DeepFace API espera esses nomes)
Model    string `json:"model_name"`
Detector string `json:"detector_backend"`
```

**Arquivo**: `internal/provider/deepface/models.go`

**Referência**: [DeepFace API Docs](https://github.com/serengil/deepface)

---

## Padrões Aprendidos

### 1. Refs para Callbacks Circulares (React/Preact)

Quando callbacks em `useCallback` se referenciam mutuamente:

```typescript
// Criar ref
const funcBRef = useRef<() => void>(() => {});

// Atualizar ref após definir a função
funcBRef.current = funcB;

// Em funcA, usar ref.current() ao invés de funcB()
funcBRef.current();
```

### 2. Wait-for-Neutral em Sequências de Movimentos

Ao detectar movimentos sequenciais (virar esquerda depois de virar direita):
1. Detectar fim do movimento anterior
2. Esperar usuário voltar à posição neutra
3. Só então iniciar detecção do próximo movimento

### 3. DeepFace Configuration

Sempre usar:
- `enforce_detection: false` para permitir detecção com baixa confiança
- `model_name` (não `model`)
- `detector_backend` (não `detector`)

---

## Checklist de Debug para Problemas Similares

### Widget não avança para próxima etapa:
- [ ] Verificar console do browser por erros
- [ ] Verificar se callbacks estão sendo chamadas (adicionar console.log)
- [ ] Verificar se há race conditions em useCallback/useState
- [ ] Considerar usar refs para callbacks que se referenciam

### Liveness não detecta movimento:
- [ ] Verificar valor de `baseYawRef` no momento certo
- [ ] Verificar se threshold está correto
- [ ] Verificar se usuário voltou à posição neutra
- [ ] Adicionar logs para ver valores de yaw em tempo real

### DeepFace retorna erro:
- [ ] Verificar nomes dos campos JSON (model_name, detector_backend)
- [ ] Verificar se enforce_detection está false
- [ ] Verificar se imagem é base64 válido **COM PREFIXO data:image/jpeg;base64,**
- [ ] Testar diretamente com curl contra DeepFace

---

### 10. DeepFace tratando base64 como caminho de arquivo

**Problema**: API retornava 500 com erro `"Confirm that /9j/4AAQSkZ... exists"`.

**Causa**: O DeepFace API interpreta o campo `img` de três formas:
1. Caminho de arquivo no servidor
2. URL HTTP/HTTPS
3. String base64 **COM prefixo `data:image/...;base64,`**

Quando enviamos base64 puro (sem prefixo), o DeepFace assume que é um caminho de arquivo e tenta abrir `/9j/4AAQSkZ...` no disco, que obviamente não existe.

**Sintoma enganoso**: O erro menciona "file not found", mas o problema é formato incorreto.

**Solução**:

```go
// ❌ ERRADO - DeepFace interpreta como caminho de arquivo
imageBase64 := base64.StdEncoding.EncodeToString(image)

// ✅ CORRETO - Prefixo data URL indica que é base64
imageBase64 := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(image)
```

**Arquivos corrigidos**:
- `internal/provider/deepface/provider.go` - DetectFaces e IndexFace

**Lição aprendida**: APIs externas podem ter comportamentos não documentados. Quando um erro parece não fazer sentido ("arquivo não encontrado" para uma string base64), investigar o **contrato real da API**, não apenas a documentação.

---

## Resumo: Anatomia do Debug

Este bug demonstra a importância de **análise de causa raiz**:

| Etapa | O que parecia | O que era |
|-------|---------------|-----------|
| 1 | "DeepFace rejeitando face" | enforce_detection=true (parcial) |
| 2 | "Backend antigo rodando" | Porta ocupada (sintoma) |
| 3 | "enforce_detection não funcionou" | Correção não aplicada (porta) |
| 4 | **CAUSA RAIZ** | Base64 sem prefixo data URL |

**Fluxo de investigação correto**:
```
Frontend → Handler → Service → Provider → Client → DeepFace
         ↓
    Ler CADA camada
         ↓
    Testar CADA camada isoladamente
         ↓
    Identificar ONDE o dado muda de formato
```
