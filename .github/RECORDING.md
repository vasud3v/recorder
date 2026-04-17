# 24/7 Recording Setup

## Quick Start

1. Edit `.github/channels.json` to add channels
2. Add secrets in repo settings (optional):
   - `CHATURBATE_COOKIES`
   - `USER_AGENT`
3. Push to GitHub or manually trigger workflow

## Edge Cases Handled

### ✅ Empty channels.json
- Workflow skips recording if no channels defined
- Won't fail or loop infinitely

### ✅ Missing channels.json
- Gracefully handles missing file
- Outputs empty array instead of failing

### ✅ Branch detection
- Auto-detects current branch for restart
- Works on any branch, not just `main`

### ✅ Rate limiting
- 30-second delay before restart
- Prevents GitHub API rate limit errors

### ✅ Restart failures
- Uses `continue-on-error` to prevent workflow failure
- Logs error but doesn't stop execution

### ✅ Concurrent runs
- GitHub prevents duplicate scheduled runs
- Manual triggers queue properly

### ✅ Artifact naming conflicts
- Uses `${{ github.run_number }}` for unique names
- Each run creates separate artifacts

### ✅ Cache conflicts
- Separate cache per channel username
- Uses `run_id` for versioning

## Limitations

### GitHub Actions Limits (Free Tier)
- **6-hour max runtime** per job → workflow restarts every 5h
- **20 concurrent jobs** → max 20 channels in parallel
- **2000 minutes/month** → ~40 hours total (not truly 24/7)
- **500MB artifacts** → use GoFile upload to save space

### Paid Tier
- **72-hour max runtime** → can extend timeout
- **60 concurrent jobs** → more parallel channels
- **3000 minutes/month** → ~60 hours total

## Workarounds

### For true 24/7:
1. **Use multiple repos** - each gets 2000 min/month
2. **Self-hosted runner** - no time limits
3. **External VPS** - run Docker container instead

### For more channels:
- Split into multiple workflow files
- Each file handles 20 channels max

### For storage:
- Enable `--enable-gofile-upload` (already set)
- Deletes local files after upload
- Database stored as artifacts

## Monitoring

Check workflow status:
```bash
# View runs
https://github.com/YOUR_USERNAME/YOUR_REPO/actions

# Download database artifacts
Actions → Select run → Artifacts section
```

## Troubleshooting

**Workflow not starting:**
- Check if Actions enabled in repo settings
- Verify `channels.json` syntax with `jq`

**Restart loop fails:**
- Check GitHub token permissions
- Ensure `workflow_dispatch` trigger exists

**Out of minutes:**
- Check usage: Settings → Billing → Actions
- Consider self-hosted runner

**Channels not recording:**
- Verify channel usernames are correct
- Check if cookies/user-agent needed
- Review job logs for errors
