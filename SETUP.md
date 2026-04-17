# Automatic 24/7 Recording Setup

## How to Use (Super Simple)

### 1. Add Streamers to Record

Edit `.github/channels.json`:

```json
{
  "channels": [
    {
      "username": "streamer1",
      "site": "chaturbate"
    },
    {
      "username": "streamer2",
      "site": "stripchat"
    },
    {
      "username": "streamer3",
      "site": "chaturbate"
    }
  ]
}
```

**That's it!** Just add username and site. Everything else is automatic.

### 2. Push to GitHub

```bash
git add .github/channels.json
git commit -m "Add streamers"
git push
```

**Done!** Recording starts automatically.

## What Happens Automatically

✅ **Highest quality** - Always records at maximum available resolution (up to 4K) and 60fps  
✅ **Auto-upload** - Uploads to GoFile.io and deletes local files  
✅ **Auto-restart** - Runs continuously 24/7 (restarts every 5 hours)  
✅ **Parallel recording** - Records up to 20 streamers simultaneously  
✅ **Auto-remux** - Fixes video seeking with FFmpeg  
✅ **Database backup** - Saves download links as artifacts  
✅ **State persistence** - Remembers progress between restarts  

## Optional: Bypass Cloudflare

If you get blocked, add these secrets in GitHub repo settings:

**Settings → Secrets and variables → Actions → New repository secret**

1. `CHATURBATE_COOKIES` - Your `cf_clearance` cookie
2. `USER_AGENT` - Your browser user agent

See [README.md](README.md#-bypass-cloudflare) for how to get these.

## Monitoring

**View recordings:**
- Go to: `https://github.com/YOUR_USERNAME/YOUR_REPO/actions`
- Click on latest workflow run
- Check logs for each streamer

**Download database:**
- Actions → Select run → Scroll to "Artifacts"
- Download `db-USERNAME-XXXX.zip`
- Contains GoFile download links

## How to Stop

**Stop all recording:**
```bash
# Delete all channels
echo '{"channels":[]}' > .github/channels.json
git add .github/channels.json
git commit -m "Stop recording"
git push
```

**Stop specific streamer:**
Just remove them from `channels.json` and push.

## Troubleshooting

**Not recording?**
- Check Actions tab for errors
- Verify streamer username is correct
- Check if you need Cloudflare bypass

**Out of GitHub minutes?**
- Free tier: 2000 min/month (~40 hours)
- Paid tier: 3000 min/month (~60 hours)
- Solution: Use multiple repos or self-hosted runner

**Need more than 20 streamers?**
- GitHub free tier limits to 20 concurrent jobs
- Split into multiple repos

## Quality Settings Explained

The workflow uses:
- `-resolution 9999` - Picks highest available (usually 1080p or 4K)
- `-framerate 60` - Prefers 60fps, falls back to 30fps if unavailable
- `--finalize-mode remux` - Fixes video seeking without re-encoding
- `--enable-gofile-upload` - Auto-uploads and deletes local files

You don't need to change anything - it's already optimized for best quality!
