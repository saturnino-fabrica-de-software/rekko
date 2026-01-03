# /benchmark - Performance Profiling & Benchmarking

Ferramenta de profiling e benchmarking para garantir que Rekko atinja os targets de performance (P99 < 5ms).

**Filosofia**: "Measure first, optimize second. No premature optimization!"
**Input**: Endpoint, função, ou package para analisar
**Output**: Relatório com métricas, flamegraph, e recomendações
**UX**: Interativo com opções de profundidade

---

## 🎯 Targets de Performance (Rekko)

| Métrica | Target | Critical |
|---------|--------|----------|
| P50 Latency | < 2ms | < 5ms |
| P99 Latency | < 5ms | < 10ms |
| P999 Latency | < 10ms | < 20ms |
| Throughput | > 10k req/s | > 5k req/s |
| Memory per request | < 1KB | < 5KB |
| GC Pause | < 1ms | < 5ms |
| Allocs per request | < 10 | < 50 |

---

## 📋 Modos de Operação

### 1. `scan` - Análise Rápida
```bash
/benchmark scan
# → Identifica hot paths
# → Lista top 10 funções mais lentas
# → Mostra allocation hotspots
```

### 2. `profile` - Profiling Detalhado
```bash
/benchmark profile
# → CPU profile com pprof
# → Memory profile
# → Goroutine analysis
# → Block profile (mutex contention)
```

### 3. `bench` - Benchmark de Função
```bash
/benchmark bench
# → Roda go test -bench
# → Compara com baseline
# → Detecta regressões
```

### 4. `load` - Teste de Carga
```bash
/benchmark load
# → Teste com k6 ou vegeta
# → Simula N usuários concorrentes
# → Mede latência sob carga
```

### 5. `compare` - Comparação A/B
```bash
/benchmark compare
# → Compara dois commits/branches
# → Mostra diff de performance
# → Detecta regressões
```

---

## 🔧 Ferramentas Utilizadas

### Go Native
```bash
# CPU Profile
go test -cpuprofile=cpu.prof -bench=. ./...

# Memory Profile
go test -memprofile=mem.prof -bench=. ./...

# Trace
go test -trace=trace.out -bench=. ./...

# Benchmark
go test -bench=. -benchmem -count=5 ./...
```

### pprof Analysis
```bash
# Interactive
go tool pprof cpu.prof

# Web UI
go tool pprof -http=:8080 cpu.prof

# Top functions
go tool pprof -top cpu.prof

# Flamegraph
go tool pprof -svg cpu.prof > cpu.svg
```

### Load Testing
```bash
# k6
k6 run --vus 100 --duration 30s scripts/load-test.js

# vegeta
echo "GET http://localhost:8080/api/v1/faces" | vegeta attack -rate=1000 -duration=30s | vegeta report
```

---

## 📊 Benchmark Template

```go
// internal/service/face_bench_test.go
package service

import (
    "testing"
)

func BenchmarkFaceSearch(b *testing.B) {
    // Setup
    svc := setupTestService(b)
    embedding := generateTestEmbedding()

    b.ResetTimer()
    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        _, err := svc.Search(testCtx, "tenant-1", embedding, 0.8, 10)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkFaceSearch_Parallel(b *testing.B) {
    svc := setupTestService(b)
    embedding := generateTestEmbedding()

    b.ResetTimer()
    b.ReportAllocs()

    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            _, _ = svc.Search(testCtx, "tenant-1", embedding, 0.8, 10)
        }
    })
}

// Benchmark with different sizes
func BenchmarkFaceSearch_10K(b *testing.B)   { benchmarkFaceSearchN(b, 10_000) }
func BenchmarkFaceSearch_100K(b *testing.B)  { benchmarkFaceSearchN(b, 100_000) }
func BenchmarkFaceSearch_1M(b *testing.B)    { benchmarkFaceSearchN(b, 1_000_000) }

func benchmarkFaceSearchN(b *testing.B, n int) {
    svc := setupTestServiceWithN(b, n)
    embedding := generateTestEmbedding()

    b.ResetTimer()
    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        _, _ = svc.Search(testCtx, "tenant-1", embedding, 0.8, 10)
    }
}
```

---

## 📈 Load Test Script (k6)

```javascript
// scripts/load-test.js
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const errorRate = new Rate('errors');
const searchLatency = new Trend('search_latency');

export const options = {
  stages: [
    { duration: '30s', target: 50 },   // Ramp up
    { duration: '1m', target: 100 },   // Sustain
    { duration: '30s', target: 200 },  // Peak
    { duration: '30s', target: 0 },    // Ramp down
  ],
  thresholds: {
    'http_req_duration': ['p(99)<5'],  // P99 < 5ms
    'errors': ['rate<0.01'],           // Error rate < 1%
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_KEY = __ENV.API_KEY || 'test-key';

export default function () {
  const headers = {
    'Content-Type': 'application/json',
    'X-API-Key': API_KEY,
    'X-Tenant-ID': 'tenant-1',
  };

  // Face search request
  const embedding = generateRandomEmbedding(512);
  const payload = JSON.stringify({
    embedding: embedding,
    threshold: 0.8,
    limit: 10,
  });

  const start = Date.now();
  const res = http.post(`${BASE_URL}/api/v1/faces/search`, payload, { headers });
  const latency = Date.now() - start;

  searchLatency.add(latency);
  errorRate.add(res.status !== 200);

  check(res, {
    'status is 200': (r) => r.status === 200,
    'latency < 5ms': () => latency < 5,
    'has results': (r) => JSON.parse(r.body).results !== undefined,
  });

  sleep(0.1); // 10 req/s per VU
}

function generateRandomEmbedding(size) {
  return Array.from({ length: size }, () => Math.random() * 2 - 1);
}
```

---

## 📋 Relatório de Benchmark

```markdown
# 📊 Benchmark Report

**Date**: 2024-01-15
**Commit**: abc1234
**Branch**: main

## 🎯 Summary

| Metric | Value | Target | Status |
|--------|-------|--------|--------|
| P50 Latency | 1.2ms | < 2ms | ✅ |
| P99 Latency | 4.8ms | < 5ms | ✅ |
| P999 Latency | 8.2ms | < 10ms | ✅ |
| Throughput | 12.5k req/s | > 10k | ✅ |
| Allocs/op | 8 | < 10 | ✅ |
| Bytes/op | 856 | < 1024 | ✅ |

## 🔥 Hot Paths

1. `FaceSearchRepository.FindSimilar` - 45% CPU
2. `pgvector.cosineSimilarity` - 30% CPU
3. `json.Marshal` - 10% CPU

## 💾 Memory Hotspots

1. `[]float64` allocations in embedding parsing
2. `sql.Rows` not properly closed in 2 locations
3. `sync.Pool` not used for frequent objects

## 📈 Recommendations

1. **Use sync.Pool for embeddings**
   - Current: 8 allocs/op
   - Expected: 2 allocs/op
   - Impact: -75% allocations

2. **Batch pgvector queries**
   - Current: 1 query per face
   - Expected: Batch of 100
   - Impact: -80% latency for multi-search

3. **Pre-allocate result slices**
   - Current: append() growing slice
   - Expected: make([]Result, 0, limit)
   - Impact: -50% allocations

## 📊 Comparison with Previous Run

| Metric | Previous | Current | Delta |
|--------|----------|---------|-------|
| P99 Latency | 5.2ms | 4.8ms | -7.7% ✅ |
| Throughput | 11.2k | 12.5k | +11.6% ✅ |
| Allocs/op | 12 | 8 | -33% ✅ |
```

---

## 🔄 CI Integration

```yaml
# .github/workflows/benchmark.yml
name: Benchmark

on:
  pull_request:
    branches: [main]

jobs:
  benchmark:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Run benchmarks
        run: |
          go test -bench=. -benchmem -count=5 ./... | tee benchmark.txt

      - name: Compare with baseline
        uses: benchmark-action/github-action-benchmark@v1
        with:
          tool: 'go'
          output-file-path: benchmark.txt
          fail-on-alert: true
          alert-threshold: '150%'  # Fail if 50% slower
          comment-on-alert: true
          github-token: ${{ secrets.GITHUB_TOKEN }}
```

---

## 💡 Exemplo de Uso

```bash
# Modo interativo (recomendado)
/benchmark
# → Claude pergunta: Qual operação? (scan/profile/bench/load/compare)
# → Claude pergunta: Qual escopo? (endpoint/função/package)
# → Execução e relatório

# Modo direto
/benchmark scan
/benchmark profile internal/service/face.go
/benchmark bench FaceSearch
/benchmark load /api/v1/faces/search --vus 100
/benchmark compare main..feat/optimization
```

---

## ✅ Checklist de Performance

Antes de mergear código performance-critical:

- [ ] Benchmark criado para função principal
- [ ] P99 < 5ms verificado
- [ ] Allocations/op medidas e aceitáveis
- [ ] Nenhum memory leak (pprof heap)
- [ ] Nenhuma goroutine leak (pprof goroutine)
- [ ] Load test passou com 100+ VUs
- [ ] Comparação com baseline não mostra regressão
- [ ] Hot paths documentados
