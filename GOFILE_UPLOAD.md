# GoFile Upload Feature

This feature automatically uploads completed recordings to GoFile.io, stores the download links in a database, and deletes the local files to save disk space.

## How It Works

1. **Recording**: Videos are recorded as usual
2. **Finalization**: After recording completes, the video is processed (if FFmpeg finalization is enabled)
3. **Upload**: The video is uploaded to GoFile.io
4. **Database**: The GoFile download link is saved to `./conf/videos.json`
5. **Cleanup**: The local video file is deleted to free up disk space

## Configuration

### Enable via CLI

```bash
./goondvr --enable-gofile-upload
```

### Enable via Web UI

1. Open the web interface at `http://localhost:8080`
2. Go to Settings
3. Check "Enable GoFile Upload"
4. Save settings

### Enable in settings.json

Edit `./conf/settings.json` and add:

```json
{
  "enable_gofile_upload": true
}
```

## API Endpoints

### Get All Uploaded Videos

```bash
GET /api/videos
```

Returns all uploaded video records:

```json
[
  {
    "id": "username_1234567890",
    "username": "example_user",
    "site": "chaturbate",
    "filename": "example_user_2026-04-17_12-30-00.mp4",
    "uploaded_at": "2026-04-17T12:35:00Z",
    "gofile_link": "https://gofile.io/d/abc123",
    "duration": 1234.5,
    "filesize_bytes": 123456789
  }
]
```

### Get Videos by Username

```bash
GET /api/videos/:username
```

Example:
```bash
curl http://localhost:8080/api/videos/example_user
```

## Database

Video records are stored in `./conf/videos.json` with the following structure:

- **id**: Unique identifier (username_timestamp)
- **username**: Channel username
- **site**: Recording site (chaturbate or stripchat)
- **filename**: Original filename
- **uploaded_at**: Upload timestamp
- **gofile_link**: GoFile download URL
- **duration**: Video duration in seconds
- **filesize_bytes**: File size in bytes

## Important Notes

1. **Disk Space**: Local files are deleted after successful upload. Make sure you have the GoFile link saved before the upload completes.

2. **Upload Failures**: If upload fails, the local file is kept and moved to the completed directory as usual.

3. **GoFile Limits**: GoFile.io has file size and bandwidth limits. Check their terms of service.

4. **No Authentication**: GoFile uploads are anonymous. Anyone with the link can download the video.

5. **Link Expiration**: GoFile links may expire after a period of inactivity. Check GoFile's retention policy.

## Troubleshooting

### Upload Failed

If you see "gofile upload failed" in the logs:
- Check your internet connection
- Verify the file exists and is readable
- Check GoFile.io status
- The local file will be kept in the completed directory

### Database Not Saving

If records aren't appearing in `/api/videos`:
- Check file permissions on `./conf/videos.json`
- Look for errors in the application logs
- Verify the upload completed successfully

## Example Usage

### CLI Mode with GoFile Upload

```bash
# Record a channel and upload to GoFile
./goondvr -u username --enable-gofile-upload

# With custom settings
./goondvr -u username \
  --enable-gofile-upload \
  --finalize-mode remux \
  --ffmpeg-container mp4
```

### Docker with GoFile Upload

```bash
docker run -d \
  --name my-dvr \
  -p 8080:8080 \
  -v "./videos:/usr/src/app/videos" \
  -v "./conf:/usr/src/app/conf" \
  -e ENABLE_GOFILE_UPLOAD=true \
  goondvr
```

### Retrieve All Videos

```bash
# Get all uploaded videos
curl http://localhost:8080/api/videos | jq

# Get videos for specific user
curl http://localhost:8080/api/videos/username | jq

# Count total uploads
curl -s http://localhost:8080/api/videos | jq 'length'

# Get total uploaded size
curl -s http://localhost:8080/api/videos | jq '[.[].filesize_bytes] | add'
```

## Integration Examples

### Python Script to Download All Videos

```python
import requests
import json

# Get all video records
response = requests.get('http://localhost:8080/api/videos')
videos = response.json()

for video in videos:
    print(f"Username: {video['username']}")
    print(f"Link: {video['gofile_link']}")
    print(f"Size: {video['filesize_bytes'] / 1024 / 1024:.2f} MB")
    print("---")
```

### Backup Database

```bash
# Backup the video database
cp ./conf/videos.json ./conf/videos.backup.json

# Restore from backup
cp ./conf/videos.backup.json ./conf/videos.json
```
