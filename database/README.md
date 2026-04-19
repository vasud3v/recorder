# Database Structure

This directory contains all recording metadata, upload history, and channel statistics.

## Directory Structure

```
database/
├── README.md                          # This file
├── backups/                           # Manual backups and exports
│   ├── channels_backup_*.json        # Channel configuration backups
│   ├── settings_backup_*.json        # Settings backups
│   └── uploaded_links.json           # Python upload script records
├── <username>/                        # Per-channel database
│   ├── uploads.json                  # Upload history for this channel
│   ├── stats.json                    # Channel statistics
│   └── metadata.json                 # Channel metadata
└── global/                            # Global database files
    ├── all_uploads.json              # Combined upload history
    ├── statistics.json               # Global statistics
    └── channels.json                 # All channels metadata

```

## File Formats

### uploads.json (Per-Channel)
```json
{
  "channel": {
    "username": "username",
    "site": "chaturbate",
    "first_recorded": "2026-04-19T20:00:00Z",
    "last_recorded": "2026-04-19T21:00:00Z"
  },
  "records": [
    {
      "id": "username_1713556800_12345",
      "username": "username",
      "site": "chaturbate",
      "filename": "username_2026-04-19_20-00-00.mp4",
      "gofile_link": "https://gofile.io/d/xxxxx",
      "uploaded_at": "2026-04-19T20:30:00Z",
      "filesize_bytes": 39000000,
      "duration_seconds": 3600,
      "resolution": "1080p",
      "framerate": 30,
      "upload_duration": 120,
      "upload_speed": 2.5,
      "convert_duration": 45,
      "status": "uploaded"
    }
  ],
  "summary": {
    "total_recordings": 1,
    "total_size_bytes": 39000000,
    "total_size_gb": 0.04,
    "total_duration_seconds": 3600,
    "total_duration_hours": 1.0,
    "average_filesize_mb": 37.2,
    "average_duration_minutes": 60
  }
}
```

### stats.json (Per-Channel)
```json
{
  "username": "username",
  "site": "chaturbate",
  "statistics": {
    "total_recordings": 10,
    "total_size_gb": 5.2,
    "total_duration_hours": 15.5,
    "first_recording": "2026-04-01T00:00:00Z",
    "last_recording": "2026-04-19T21:00:00Z",
    "average_session_minutes": 93,
    "longest_session_minutes": 180,
    "shortest_session_minutes": 30,
    "recordings_per_day": 2.5,
    "offline_streak": 0,
    "last_online": "2026-04-19T21:00:00Z"
  },
  "quality": {
    "resolutions": {
      "1080p": 8,
      "720p": 2
    },
    "framerates": {
      "30": 7,
      "60": 3
    },
    "average_bitrate_mbps": 5.2
  },
  "uploads": {
    "total_uploaded": 10,
    "failed_uploads": 0,
    "average_upload_speed_mbps": 2.5,
    "total_upload_time_minutes": 45
  }
}
```

### metadata.json (Per-Channel)
```json
{
  "username": "username",
  "site": "chaturbate",
  "display_name": "Username",
  "profile_url": "https://chaturbate.com/username/",
  "added_at": "2026-04-01T00:00:00Z",
  "last_updated": "2026-04-19T21:00:00Z",
  "settings": {
    "resolution": 1080,
    "framerate": 30,
    "max_duration": 45,
    "interval": 1,
    "enabled": true,
    "paused": false
  },
  "tags": ["favorite", "daily"],
  "notes": "Optional notes about this channel"
}
```

## Usage

### GitHub Actions Workflow
The workflow automatically:
1. Records streams for each channel
2. Converts to MP4 and uploads to GoFile
3. Updates `database/<username>/uploads.json`
4. Commits changes back to the repository

### Local Usage
Use the Python scripts to manage the database:
```bash
# Upload a video and update database
python upload_to_gofile.py videos/completed/video.mp4

# View statistics
python scripts/view_stats.py username

# Export database
python scripts/export_database.py --format csv
```

## Backup Strategy

1. **Automatic Backups**: GitHub Actions commits database changes after each run
2. **Manual Backups**: Use `database/backups/` for manual exports
3. **Git History**: Full history available in git commits

## Maintenance

- Database files are automatically cleaned and deduplicated
- Old records are never deleted (append-only)
- Statistics are recalculated on each update
- Backups are timestamped for easy restoration
