# Timeout & Configuration Issues Analysis

## Issues Found

### ✅ Issue #1: Recording Timeout Doesn't Match max_duration (FIXED)
**Problem**: Hardcoded 30-minute timeout regardless of user's `max_duration` setting.

**Example**:
- User sets `max_duration: 15`
- Workflow uses `timeout 30m` (hardcoded)
- Recording continues for 30 minutes instead of 15

**Fix Applied**:
```bash
RECORDING_TIMEOUT=$((MAX_DURATION + 5))
timeout ${RECORDING_TIMEOUT}m ./goondvr ...
```

**Result**: 
- 15 min duration → 20 min timeout
- 45 min duration → 50 min timeout
- 240 min duration → 245 min timeout

---

### ⚠️ Issue #2: Job Timeout vs Recording Duration Mismatch
**Problem**: Job timeout (330 min) might not be enough for large `max_duration` values.

**Current State**:
```yaml
timeout-minutes: 330  # 5.5 hours
```

**Scenarios**:
- `max_duration: 240` (4 hours) → OK
- `max_duration: 240` + multiple chunks → Might exceed 330 min
- Background monitor: 6 hours max → Exceeds job timeout!

**Risk**: Job killed before monitor timeout triggers

**Recommendation**: Adjust job timeout based on max_duration

**Proposed Fix**:
```yaml
# For max_duration up to 240 minutes
timeout-minutes: 330  # Current

# But monitor timeout (6 hours = 360 min) exceeds this!
# Should be: timeout-minutes: 390  # 6.5 hours
```

---

### ⚠️ Issue #3: Monitor Timeout Exceeds Job Timeout
**Problem**: Background monitor can run for 6 hours, but job times out at 5.5 hours.

**Current State**:
```bash
MAX_MONITOR_TIME=$((6 * 3600))  # 6 hours
```

**Job timeout**:
```yaml
timeout-minutes: 330  # 5.5 hours
```

**Result**: Job killed before monitor can gracefully stop

**Recommendation**: Align timeouts

**Proposed Fix**:
```bash
# Option 1: Reduce monitor timeout
MAX_MONITOR_TIME=$((5 * 3600))  # 5 hours (safer)

# Option 2: Increase job timeout
timeout-minutes: 390  # 6.5 hours
```

---

### ⚠️ Issue #4: Processing Job Timeout Too Long
**Problem**: Processing job has 360-minute (6 hour) timeout, but most processing should be faster.

**Current State**:
```yaml
timeout-minutes: 360  # 6 hours
```

**Typical Processing Time**:
- 50 files × (2 min convert + 10 min upload) = 600 minutes (10 hours!)
- This exceeds the timeout!

**Risk**: Large batches will timeout

**Recommendation**: Process fewer files or increase timeout

**Proposed Fix**:
```yaml
# Option 1: Reduce file limit
COMPLETED_FILES=$(find ... | head -25)  # 25 instead of 50

# Option 2: Increase timeout
timeout-minutes: 720  # 12 hours

# Option 3: Add per-file timeout check
PROCESSING_START=$(date +%s)
MAX_PROCESSING_TIME=$((5 * 3600))  # 5 hours

for MARKER in $COMPLETED_FILES; do
  ELAPSED=$(($(date +%s) - PROCESSING_START))
  if [ $ELAPSED -gt $MAX_PROCESSING_TIME ]; then
    echo "⏱️  Processing time limit reached - stopping"
    break
  fi
  # ... process file
done
```

---

### ⚠️ Issue #5: FFmpeg Timeout Too Short for Large Files
**Problem**: 30-minute timeout for FFmpeg conversion might not be enough for very large files.

**Current State**:
```bash
timeout 30m ffmpeg -nostdin -y -i "$VIDEO_FILE" ...
```

**Scenarios**:
- 240-minute recording at high quality = ~10GB file
- Conversion might take >30 minutes on slow runners

**Risk**: Conversion fails on large files

**Recommendation**: Dynamic timeout based on file size

**Proposed Fix**:
```bash
# Calculate timeout based on file size
FILE_SIZE_GB=$(echo "scale=2; $VIDEO_SIZE_BYTES / 1024 / 1024 / 1024" | bc)
CONVERT_TIMEOUT=30  # Default 30 minutes

# Add 10 minutes per GB over 2GB
if (( $(echo "$FILE_SIZE_GB > 2" | bc -l) )); then
  EXTRA_TIME=$(echo "($FILE_SIZE_GB - 2) * 10" | bc | cut -d. -f1)
  CONVERT_TIMEOUT=$((30 + EXTRA_TIME))
  echo "Large file detected - using ${CONVERT_TIMEOUT} minute timeout"
fi

timeout ${CONVERT_TIMEOUT}m ffmpeg ...
```

---

### ⚠️ Issue #6: Upload Timeout Too Short for Large Files
**Problem**: 60-minute timeout might not be enough for very large files on slow connections.

**Current State**:
```bash
timeout 60m curl -X POST "https://${SERVER}.gofile.io/contents/uploadfile" ...
```

**Scenarios**:
- 10GB file at 2MB/s = 83 minutes
- Exceeds 60-minute timeout!

**Risk**: Large file uploads fail

**Recommendation**: Dynamic timeout based on file size

**Proposed Fix**:
```bash
# Calculate timeout based on file size (assume 2MB/s minimum speed)
FILE_SIZE_MB=$(echo "scale=0; $MP4_SIZE_BYTES / 1024 / 1024" | bc)
UPLOAD_TIMEOUT=$((FILE_SIZE_MB / 2 / 60 + 10))  # Size/2MB/s/60s + 10min buffer

# Minimum 60 minutes, maximum 180 minutes
if [ $UPLOAD_TIMEOUT -lt 60 ]; then
  UPLOAD_TIMEOUT=60
elif [ $UPLOAD_TIMEOUT -gt 180 ]; then
  UPLOAD_TIMEOUT=180
  echo "⚠️  File very large - upload may take up to 3 hours"
fi

echo "Upload timeout: ${UPLOAD_TIMEOUT} minutes"
timeout ${UPLOAD_TIMEOUT}m curl ...
```

---

### ⚠️ Issue #7: Inconsistent Timeout Messages
**Problem**: Some timeout messages reference old hardcoded values.

**Current Code**:
```bash
echo "⏱️  30-minute check timeout - continuing..."
```

**Issue**: This message appears even when timeout is not 30 minutes

**Fix**: Use variable in message
```bash
echo "⏱️  ${RECORDING_TIMEOUT}-minute check timeout - continuing..."
```

---

### ⚠️ Issue #8: No Timeout on Database Operations
**Problem**: Database lock has 30-second timeout, but jq operations have no timeout.

**Current State**:
```bash
flock -x -w 30 200  # Has timeout
jq ".records += [$RECORD]" database/uploads.json  # No timeout!
```

**Risk**: Large database files could cause jq to hang

**Recommendation**: Add timeout to jq operations

**Proposed Fix**:
```bash
# Add timeout to jq
if timeout 30s jq ".records += [$RECORD]" database/uploads.json > database/uploads.json.tmp 2>/dev/null; then
  mv database/uploads.json.tmp database/uploads.json
else
  echo "✗ Database operation timed out"
  rm -f database/uploads.json.tmp
fi
```

---

### ⚠️ Issue #9: FlareSolverr Test Timeout Too Short
**Problem**: 30-second timeout might not be enough for FlareSolverr to respond.

**Current State**:
```bash
curl -s --max-time 30 -X POST http://localhost:8191/v1 ...
```

**FlareSolverr Request**:
```json
{"maxTimeout":60000}  // 60 seconds requested
```

**Conflict**: curl times out at 30s, but request asks for 60s

**Recommendation**: Align timeouts

**Proposed Fix**:
```bash
# Increase curl timeout to match request timeout + buffer
curl -s --max-time 90 -X POST http://localhost:8191/v1 \
  -H "Content-Type: application/json" \
  -d '{"cmd":"request.get","url":"https://chaturbate.com","maxTimeout":60000}' | jq .
```

---

### ⚠️ Issue #10: No Timeout on Artifact Operations
**Problem**: Artifact upload/download has no explicit timeout.

**Current State**:
```yaml
- uses: actions/upload-artifact@v4
  # No timeout specified
```

**Risk**: Very large artifacts could hang indefinitely

**Recommendation**: Add timeout or size check

**Proposed Fix**:
```yaml
- name: Upload with timeout check
  timeout-minutes: 60  # Add step-level timeout
  uses: actions/upload-artifact@v4
  ...
```

---

## Summary of Timeout Values

### Current Timeouts
| Operation | Current | Issue |
|-----------|---------|-------|
| Record job | 330 min | ⚠️ Less than monitor (360 min) |
| Process job | 360 min | ⚠️ Too short for 50 files |
| Recording | 30 min | ✅ Now dynamic (max_duration + 5) |
| Monitor | 360 min | ⚠️ Exceeds job timeout |
| FFmpeg | 30 min | ⚠️ Too short for large files |
| Upload | 60 min | ⚠️ Too short for large files |
| FlareSolverr | 30s | ⚠️ Less than request timeout (60s) |
| Database lock | 30s | ✅ OK |
| jq operations | None | ⚠️ No timeout |

### Recommended Timeouts
| Operation | Recommended | Reason |
|-----------|-------------|--------|
| Record job | 390 min | Allow monitor to complete |
| Process job | 720 min | Handle 50 files safely |
| Recording | max_duration + 5 | ✅ Already fixed |
| Monitor | 300 min | Less than job timeout |
| FFmpeg | 30 + (size-2)*10 | Scale with file size |
| Upload | size/2MB/s + 10 | Scale with file size |
| FlareSolverr | 90s | Match request + buffer |
| Database lock | 30s | ✅ OK |
| jq operations | 30s | Prevent hangs |

---

## Configuration Matrix

### For 15-minute chunks:
```yaml
max_duration: 15
Recording timeout: 20 min
Job timeout: 390 min (unchanged)
Files per run: 50 (OK)
```

### For 45-minute chunks (default):
```yaml
max_duration: 45
Recording timeout: 50 min
Job timeout: 390 min (unchanged)
Files per run: 50 (OK)
```

### For 240-minute chunks:
```yaml
max_duration: 240
Recording timeout: 245 min
Job timeout: 390 min (OK)
Files per run: 25 (reduce from 50)
Processing timeout: 720 min (increase)
```

---

## Testing Checklist

- [ ] Test 15-minute chunks (should timeout at 20 min)
- [ ] Test 45-minute chunks (should timeout at 50 min)
- [ ] Test 240-minute chunks (should timeout at 245 min)
- [ ] Test large file conversion (>5GB)
- [ ] Test large file upload (>5GB)
- [ ] Test processing 50 files
- [ ] Test job timeout (should complete before 390 min)
- [ ] Test monitor timeout (should stop before job timeout)
- [ ] Test FlareSolverr timeout
- [ ] Test database operations with large files

---

## Priority Fixes

### Critical (Fix Now)
1. ✅ Recording timeout mismatch (FIXED)
2. ⚠️ Monitor timeout exceeds job timeout
3. ⚠️ Processing job timeout too short

### High Priority
4. ⚠️ FFmpeg timeout too short for large files
5. ⚠️ Upload timeout too short for large files
6. ⚠️ No timeout on jq operations

### Medium Priority
7. ⚠️ Inconsistent timeout messages
8. ⚠️ FlareSolverr timeout mismatch
9. ⚠️ No timeout on artifact operations

### Low Priority
10. Documentation updates
11. Add timeout monitoring/logging
12. Add timeout configuration options
