# DADV Project - Security Architecture

## Executive Summary

This document outlines security considerations for the DADV (Data Analysis & Visualization) system. Designed with local-first deployment in mind, but architected to support secure external deployment.

---

## 1. Threat Model

### 1.1 Assets to Protect

| Asset | Sensitivity | Impact if Compromised |
|-------|------------|-------------------|
| Uploaded datasets | High | Data exposure, malicious file injection |
| Processed metadata | Medium | Information disclosure |
| Processing results | Medium | Business intelligence loss |
| System availability | High | Service denial |
| Credentials/secrets | Critical | Full system compromise |

### 1.2 Threat Actors

1. **Unauthenticated users** - External attackers scanning/random probing
2. **Authenticated users (normal)** - Legitimate users attempting privilege escalation
3. **Authenticated users (malicious)** - Compromised accounts
4. **Internal threats** - Misconfigured services, insider risk
5. **Supply chain** - Compromised dependencies

### 1.3 Attack Vectors

```
THREAT LANDSCAPE:
┌──────────────────────────────────────────────────────────────────────────┐
│                                                                  │
│  [INTERNET] ──────────────────────┐                              │
│         │                        │                              │
│         ▼                        ▼                              │
│  ┌─────────────┐          ┌─────────────┐                       │
│  │API Attacks │          │ Frontend   │                       │
│  │-Injection │          │-XSS       │                       │
│  │-DDoS      │          │-CSRF      │                       │
│  │-Auth bypass│         │-Clickjack │                       │
│  └─────┬─────┘          └─────┬─────┘                       │
│        │                      │                              │
│        ▼                      ▼                              │
│  ┌─────────────────────────────────────────┐                   │
│  │          APPLICATION LAYER              │                   │
│  │  Go API ─ Queue ─ Worker ─ Database     │                   │
│  │  - Path traversal  - Queue injection  │                   │
│  │  - File inclusion   - Memory exhaust  │                   │
│  │  - Command exec    - SQL injection   │                   │
│  └─────────────────────────────────────────┘                   │
│        │                                                       │
│        ▼                                                       │
│  ┌─────────────┐                                              │
│  │Third-Party  │───────────┐                                   │
│  │Dependencies │           │                                   │
│  │-PyPI compromise│      ▼                                        │
│  │-NPM compromise │  [COMPROMISE]                               │
│  │-Go module      │                                                │
│  └───────────────┘                                                │
└────────────────────────────────────────────────────────────────┘
```

---

## 2. Security Layers

### 2.1 Entry Point Security (API)

**Threats:**
- SQL/NoSQL injection via input fields
- Path traversal via file uploads
- Command injection
- Denial of Service (large payloads)
- Authentication bypass

**Mitigations:**
```go
// INPUT VALIDATION - gin.Bind() with struct tags
type UploadRequest struct {
    File *multipart.FileHeader `binding:"required"`
}

// FILE TYPE ALLOWLIST
var allowedTypes = map[string]bool{
    "text/csv":                          true,
    "application/json":                  true,
    "application/vnd.ms-excel":         true,
}

// SIZE LIMITS
const maxFileSize = 100 * 1024 * 1024 // 100MB

// RATE LIMITING
// - Global: 100 requests/minute per IP
// - Upload: 10 uploads/minute per user

// CORS - Strict origin allowlist
config := cors.Config{
    AllowOrigins:     []string{"http://localhost:5173"},
    AllowMethods:   []string{"GET", "POST", "OPTIONS"},
    AllowHeaders:   []string{"Origin", "Content-Type", "Accept"},
    ExposeHeaders:  []string{"Content-Length"},
    MaxAge:         86400,
}
```

### 2.2 File Upload Security

**Threats:**
- Malicious file execution (webshells, scripts)
- ZIP bombs (decompression bombs)
- Filename exploits (path traversal)
- Double extensions bypassing checks
- Polyglot files (embedded executables)

**Mitigations:**
```go
func ValidateAndSanitizeUpload(header *multipart.FileHeader) error {
    // 1. CONTENT-TYPE CHECK
    contentType := header.Header.Get("Content-Type")
    if !allowedTypes[contentType] {
        return fmt.Errorf("disallowed file type: %s", contentType)
    }

    // 2. MAGIC BYTE VERIFICATION
    magicBytes, _ := readFileMagic(header)
    if !isAllowedMagic(magicBytes) {
        return fmt.Errorf("file type mismatch")
    }

    // 3. EXTENSION SANITIZATION
    safeName := sanitizeFilename(header.Filename)
    if !isAllowedExtension(safeName) {
        return fmt.Errorf("disallowed extension")
    }

    // 4. SIZE LIMIT
    if header.Size > maxFileSize {
        return fmt.Errorf("file too large: %d > %d", header.Size, maxFileSize)
    }

    // 5. ZIP BOMB DETECTION (content-based)
    if isLikelyZipBomb(header) {
        return fmt.Errorf("potential zip bomb detected")
    }

    return nil
}

func sanitizeFilename(name string) string {
    // Remove path components, replace dangerous chars
    name = filepath.Base(name)
    replacer := strings.NewReplacer("/", "_", "\\", "_", "..", "_")
    return replacer.Replace(name)
}
```

### 2.3 Queue Security (Redis)

**Threats:**
- Queue injection (junk jobs)
- Job tampering
- Status manipulation
- Memory exhaustion (large keys)
- Connection hijacking

**Mitigations:**
```go
// 1. AUTHENTICATED REDIS CONNECTION
rdb := redis.NewClient(&redis.Options{
    Addr:         config.RedisAddr,
    Password:    config.RedisPassword,  // Set for local + production
    DB:          0,
    DialTimeout: 5 * time.Second,
})

// 2. JOB VALIDATION BEFORE ENQUEUE
func validateJobPayload(payload []byte) error {
    var job JobPayload
    if err := json.Unmarshal(payload, &job); err != nil {
        return fmt.Errorf("invalid job payload")
    }
    // Validate required fields
    if job.JobID == "" || job.FilePath == "" {
        return fmt.Errorf("missing required fields")
    }
    // Validate path is within allowed directory
    if !isPathSafe(job.FilePath) {
        return fmt.Errorf("path not allowed")
    }
    return nil
}

// 3. INPUT SANITIZATION
func sanitizeJobInput(input string) string {
    // Remove any command injection attempts
    input = strings.ReplaceAll(input, ";", "")
    input = strings.ReplaceAll(input, "|", "")
    input = strings.ReplaceAll(input, "`", "")
    input = strings.ReplaceAll(input, "$", "")
    return input
}
```

### 2.4 Database Security (SQLite)

**Threats:**
- SQL injection
- Data exfiltration
- Query denial (complex queries)
- Schema tampering

**Mitigations:**
```go
// 1. PARAMETERIZED QUERIES ONLY
query := "SELECT id, filename, file_size, file_type FROM files WHERE id = ?"
rows, err := db.QueryContext(ctx, query, fileID)

// 2. PRAGMA ENFORCEMENT
_, err = db.ExecContext(ctx, `
    PRAGMA journal_mode=WAL;
    PRAGMA synchronous=NORMAL;
    PRAGMA foreign_keys=ON;
    PRAGMA page_size=4096;
`)

// 3. LIMIT RESULTS
query := "SELECT * FROM results LIMIT 1000 OFFSET ?" // Prevent unbounded

// 4. READ-ONLY MODE FOR QUERIES
db.SetMaxOpenConns(25)
```

### 2.5 Output Security (Responses)

**Threats:**
- Information disclosure
- Stack traces in errors
- Sensitive data in logs

**Mitigations:**
```go
// CUSTOM ERROR HANDLER
func errorHandler(c *gin.Context, err error) {
    // Sanitize error messages
    var msg string
    if isDebugMode {
        msg = err.Error()  // Full in debug
    } else {
        msg = "Internal server error"  // Generic in prod
        log.Error(err)  // Log full locally
    }
    c.JSON(500, gin.H{"error": msg})
}

// RESPONSE FILTERING
func filterSensitiveData(resp map[string]interface{}) map[string]interface{} {
    filtered := make(map[string]interface{})
    for k, v := range resp {
        if sensitiveFields[k] {
            continue  // Skip sensitive fields
        }
        filtered[k] = v
    }
    return filtered
}
```

---

## 3. Third-Party Risk Analysis

### 3.1 Dependency Risks

| Dependency | Risk | Mitigation |
|-----------|------|------------|
| Go stdlib | Low | Trusted, audited |
| gin (web framework) | Low | Mature, audit-enabled |
| github.com/redis/go-redis | Low | Trusted, ACLs |
| Python packages | Medium | Hash verification, pin versions |
| React | Low | Trusted, subresource integrity |
| recharts | Low | Trusted |

### 3.2 Supply Chain Mitigations

```yaml
# go.mod
module dadv-project

go 1.21

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/redis/go-redis/v9 v9.3.0
)

# Verify dependencies
require (
    golang.org/x/crypto v0.17.0
)

# go.sum - verify checksums
```

```toml
# requirements.txt - pin with hashes
pandas==2.1.4 --hash=sha256:abc123...
numpy==1.26.2 --hash=sha256:def456...
```

### 3.3 Future External Deployment Risks

| Risk | Pre-deployment Action |
|------|----------------------|
| Redis exposed | Enable AUTH, use TLS |
| Database exposed | Network isolation, TLS |
| API exposed | Rate limiting, WAF |
| Frontend XSS | CSP headers, sanitization |
| JWT secrets | Rotate, use strong keys |

---

## 4. Security Tradeoffs

### 4.1 Local vs Production Security

| Aspect | Local (Current) | Production (Future) |
|--------|-----------------|---------------------|
| Authentication | None | JWT/API keys |
| Encryption | None | TLS 1.3 |
| Rate limiting | Minimal | Strict |
| Logging | Verbose | Structured |
| Error messages | Full | Sanitized |
| CORS | Permissive | Strict origin |

### 4.2 Usability vs Security

| Feature | Security Cost | Mitigation |
|---------|---------------|------------|
| File uploads | Execution risk | Content inspection, sandboxing |
| API access | Attack surface | Auth + rate limiting |
| Data export | Exfiltration | Access controls |
| Real-time updates | WebSocket exposure | Auth on connect |

### 4.3 Performance vs Security

| Trade-off | Impact | Recommendation |
|-----------|--------|--------------|
| Large uploads | Memory exhaustion | Stream processing |
| Complex queries | DoS risk | Query timeouts |
| Queue processing | Resource exhaustion | Job timeouts |
| Caching | Stale data | TTL limits |

---

## 5. Security Checklist

### 5.1 Pre-Deployment Audit

- [ ] Change all default credentials
- [ ] Enable Redis authentication
- [ ] Configure firewall rules
- [ ] Set up TLS certificates
- [ ] Implement API authentication
- [ ] Add rate limiting
- [ ] Configure CORS strict origins
- [ ] Set up logging/monitoring
- [ ] Test error sanitization
- [ ] Verify file validation
- [ ] Audit dependency versions
- [ ] Test injection attempts

### 5.2 Continuous Security

- [ ] Regular dependency updates
- [ ] Security patches applied
- [ ] Log review cadence
- [ ] Access audit
- [ ] Backup verification
- [ ] Incident response plan

---

## 6. Incident Response

### 6.1 Security Events

| Event | Response | Owner |
|-------|----------|-------|
| Failed login attempts | Block IP, alert | System |
| Large upload detected | Quarantine, inspect | Worker |
| Invalid job payloads | Reject, log | API |
| Database errors | Sanitize, alert | System |
| Unusual query patterns | Rate limit, alert | System |

### 6.2 Contact Points

- **API**: Returns generic errors (no stack traces)
- **Logs**: Stored locally, rotate weekly
- **Monitoring**: Future (not implemented in v1)

---

## 7. Security Summary

The DADV system is designed with security in mind:

1. **Defense in depth** - Multiple security layers
2. **Least privilege** - Minimal permissions
3. **Input validation** - Strict allowlists
4. **Output sanitization** - No sensitive data leakage
5. **Logging** - For audit and debugging
6. **Future-ready** - Prepared for external deployment

For a local-first system, current security is adequate. Before external deployment:
- Enable all authentication
- Configure TLS
- Add rate limiting
- Audit all configurations