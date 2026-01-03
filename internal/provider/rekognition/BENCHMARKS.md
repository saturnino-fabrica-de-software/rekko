# Rekognition Provider Benchmarks

## Overview

Este arquivo contém benchmarks de performance para o AWS Rekognition Provider. Os benchmarks medem o **overhead local** (não a latência de rede da AWS), utilizando mocks para simular as respostas da API.

## Target de Performance

- **P99 Latency**: < 200ms (incluindo latência de rede AWS)
- **Local Overhead**: < 1μs para operações críticas
- **Allocations**: Minimizar para evitar GC pressure

## Running Benchmarks

### Run all benchmarks
```bash
go test -bench=. -benchmem ./internal/provider/rekognition/
```

### Run specific benchmark
```bash
go test -bench=BenchmarkSearchFacesByImage -benchmem ./internal/provider/rekognition/
```

### Run with CPU profile
```bash
go test -bench=BenchmarkSearchFacesByImage -benchmem -cpuprofile=cpu.prof ./internal/provider/rekognition/
go tool pprof -http=:8080 cpu.prof
```

### Run with memory profile
```bash
go test -bench=BenchmarkSearchFacesByImage -benchmem -memprofile=mem.prof ./internal/provider/rekognition/
go tool pprof -http=:8080 mem.prof
```

### Compare benchmarks (before/after optimization)
```bash
# Before optimization
go test -bench=. -benchmem -count=5 ./internal/provider/rekognition/ | tee old.txt

# After optimization
go test -bench=. -benchmem -count=5 ./internal/provider/rekognition/ | tee new.txt

# Compare
benchstat old.txt new.txt
```

## Benchmark Results Analysis

### Current Results (Apple M1)

```
BenchmarkDetectFaces-8                             	 4052990	       325.6 ns/op	     500 B/op	      15 allocs/op
BenchmarkIndexFace-8                               	 4310176	       279.2 ns/op	     496 B/op	      14 allocs/op
BenchmarkSearchFacesByImage-8                      	 3914942	       304.6 ns/op	     504 B/op	      17 allocs/op
BenchmarkCompareFaceImages-8                       	10210875	       119.9 ns/op	     288 B/op	       7 allocs/op
BenchmarkValidateImage-8                           	574463997	         2.089 ns/op	       0 B/op	       0 allocs/op
BenchmarkCalculateQualityScore-8                   	1000000000	         0.3235 ns/op	       0 B/op	       0 allocs/op
```

### Key Findings

#### ✅ Excellent Performance
- **ValidateImage**: 2.089 ns/op, zero allocations
- **CalculateQualityScore**: 0.3235 ns/op, zero allocations

#### ⚠️ Acceptable Performance (Local Overhead)
- **DetectFaces**: 325.6 ns/op, 500 B/op, 15 allocs/op
- **IndexFace**: 279.2 ns/op, 496 B/op, 14 allocs/op
- **SearchFacesByImage**: 304.6 ns/op, 504 B/op, 17 allocs/op (CRITICAL PATH)
- **CompareFaceImages**: 119.9 ns/op, 288 B/op, 7 allocs/op

**Note**: Esses valores representam apenas o overhead local. A latência real incluirá:
- Network latency to AWS: ~50-150ms (US East)
- AWS Rekognition processing: ~100-300ms
- Total P99: Target < 200ms (requires optimization on AWS side)

## Benchmark Descriptions

### Core Operations

#### BenchmarkDetectFaces
- **Purpose**: Face detection operation
- **Usage**: Initial face validation before registration
- **Critical**: Medium (not in hot path)

#### BenchmarkIndexFace
- **Purpose**: Face registration/indexing
- **Usage**: User registration flow
- **Critical**: Medium (one-time operation per user)

#### BenchmarkSearchFacesByImage
- **Purpose**: 1:N face search in collection
- **Usage**: Event entry verification (CRITICAL PATH)
- **Critical**: **HIGH** - This is the most performance-sensitive operation
- **Target**: Local overhead < 500ns, Total P99 < 200ms

#### BenchmarkCompareFaceImages
- **Purpose**: 1:1 face comparison
- **Usage**: Verification flow
- **Critical**: Medium

#### BenchmarkDeleteFace
- **Purpose**: Face removal from collection
- **Usage**: User deletion, LGPD compliance
- **Critical**: Low

#### BenchmarkGetFaceCount
- **Purpose**: Retrieve face count from collection
- **Usage**: Monitoring, statistics
- **Critical**: Low

### Validation & Utility

#### BenchmarkValidateImage
- **Purpose**: Image validation (size, format)
- **Usage**: Pre-processing before AWS calls
- **Critical**: High (fast-fail optimization)
- **Target**: < 5ns, zero allocations ✅

#### BenchmarkCalculateQualityScore
- **Purpose**: Quality score calculation from AWS metrics
- **Usage**: Face quality assessment
- **Critical**: Medium
- **Target**: < 1ns, zero allocations ✅

### Scalability Tests

#### BenchmarkDetectFaces_ImageSizes
- **Purpose**: Impact of image size on performance
- **Sizes**: 1KB, 100KB, 1MB, 5MB
- **Finding**: Local overhead is constant (~230ns), size doesn't affect local processing

#### BenchmarkSearchFacesByImage_ResultSizes
- **Purpose**: Impact of result count on performance
- **Sizes**: 0, 1, 10, 100 results
- **Finding**:
  - 0 results: 206 ns/op, 328 B/op
  - 1 result: 225 ns/op, 376 B/op
  - 10 results: 342 ns/op, 744 B/op
  - 100 results: 1183 ns/op, 4426 B/op
  - **Conclusion**: Linear growth in allocations with result count

#### BenchmarkDetectFaces_MultipleFaces
- **Purpose**: Detection efficiency with multiple faces
- **Sizes**: 1, 5, 10, 50 faces
- **Finding**:
  - 1 face: 94.68 ns/op, 224 B/op
  - 5 faces: 148.5 ns/op, 416 B/op
  - 10 faces: 200.2 ns/op, 656 B/op
  - 50 faces: 667.1 ns/op, 2864 B/op
  - **Conclusion**: Efficient linear scaling

#### BenchmarkCompareFaceImages_VariousSimilarities
- **Purpose**: Impact of similarity threshold
- **Thresholds**: 0.5, 0.7, 0.8, 0.9, 0.95
- **Finding**: Threshold doesn't affect local performance (~120ns constant)

## Optimization Opportunities

### Current State ✅
1. **ValidateImage**: Perfect (2ns, zero allocs)
2. **CalculateQualityScore**: Perfect (0.3ns, zero allocs)
3. **Core operations**: Acceptable local overhead

### Potential Improvements
1. **Reduce allocations in SearchFacesByImage**: Currently 17 allocs/op
   - Consider object pooling for SearchResult slices
   - Pre-allocate result slices with capacity hint

2. **Reduce allocations in DetectFaces**: Currently 15 allocs/op
   - Pool BoundingBox and DetectedFace structs
   - Pre-allocate faces slice

3. **Monitor large result sets**: 100 results = 4426 B/op
   - Consider pagination for large collections
   - Implement result streaming if needed

## Real-World Performance

### Expected P99 Latency Breakdown (US East)

```
Component                  | Latency
---------------------------|----------
Local overhead             | < 1μs
Serialization (JSON)       | ~1-2ms
Network (US → AWS US-East) | ~50-150ms
AWS Rekognition Processing | ~100-300ms
---------------------------|----------
Total P99                  | ~150-450ms
Target                     | < 200ms ⚠️
```

### Optimization Strategy for Real-World P99 < 200ms

1. **Use AWS in same region as application**: Reduce network latency
2. **Enable connection pooling**: Reuse AWS SDK connections
3. **Implement caching**: Cache recent search results (with TTL)
4. **Use AWS PrivateLink**: Reduce internet routing overhead
5. **Pre-warm connections**: Keep connection pool hot
6. **Monitor AWS Rekognition limits**: Avoid throttling

## Continuous Monitoring

### Regression Testing
```bash
# Run before committing changes
make bench-rekognition

# Compare with baseline
benchstat baseline.txt new.txt
```

### CI/CD Integration
```yaml
# .github/workflows/benchmark.yml
- name: Run Benchmarks
  run: |
    go test -bench=. -benchmem ./internal/provider/rekognition/ | tee benchmark.txt
    # Upload to performance monitoring system
```

### Alert Thresholds

| Metric | Warning | Critical |
|--------|---------|----------|
| SearchFacesByImage ns/op | > 500 | > 1000 |
| SearchFacesByImage allocs | > 20 | > 30 |
| ValidateImage ns/op | > 10 | > 50 |

## Related Documentation

- [AWS Rekognition Performance Guide](https://docs.aws.amazon.com/rekognition/latest/dg/limits.html)
- [Go Benchmark Best Practices](https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go)
- [Rekko Performance Targets](/docs/performance.md)

---

**Last Updated**: 2026-01-03
**Benchmark Environment**: Apple M1, Go 1.22
