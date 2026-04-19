# Database Structure

This directory contains all recording metadata organized by streamer and date.

## Directory Structure

```
database/
├── README.md                          # This file
├── backups/                           # Manual backups and exports
│   ├── full_backup_*.json            # Complete database backups
│   ├── channels_backup_*.json        # Channel configuration backups
│   ├── settings_backup_*.json        # Settings backups
│   └── uploaded_links.json           # Python upload script records
├── global/                            # Global database files
│   └── all_stats.json                # Combined statistics for all streamers
├── <username>/                        # Per-streamer directory
│   ├── stats.json                    # Overall streamer statistics
│   ├── 2026-04-19/                   # Date-specific recordings
│   │   ├── recordings.json           # All recordings from this date
│   │   └── metadata.json             # Date metadata (notes, tags)
│   ├── 2026-04-20/
│   │   ├── recordings.json
│   │   └── metadata.json
│   └── ...
└── <another_username>/
    ├── stats.json
    ├── 2026-04-19/
    │   ├── recordings.json
    │   └── metadata.json
    └── ...
```

## File Formats

### recordings.json (Per-Date)
```json
{
  "date": "2026-04-19",
  "username": "honeyyykate",
  "recordings": [
    {
      "id": "honeyyykate_1713556800_12345",
      "username": "honeyyykate",
      "site": "chaturbate",
      "filename": "honeyyykate_2026-04-19_20-00-00.mp4",
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
    "total_size_mb": 37.2,
    "total_duration_seconds": 3600,
    "total_duration_minutes": 60.0,
    "first_recording_time": "2026-04-19T20:00:00Z",
    "last_recording_time": "2026-04-19T21:00:00Z"
  }
}
```

### metadata.json (Per-Date)
```json
{
  "date": "2026-04-19",
  "username": "honeyyykate",
  "created_at": "2026-04-19T20:00:00Z",
  "notes": "Great stream today!",
  "tags": ["long-session", "high-quality"]
}
```

### stats.json (Per-Streamer)
```json
{
  "username": "honeyyykate",
  "site": "chaturbate",
  "created_at": "2026-04-01T00:00:00Z",
  "last_updated": "2026-04-19T21:00:00Z",
  "statistics": {
    "total_recordings": 25,
    "total_size_gb": 15.5,
    "total_duration_hours": 42.5,
    "total_days_recorded": 10,
    "first_recording_date": "2026-04-01",
    "last_recording_date": "2026-04-19",
    "average_recordings_per_day": 2.5,
    "average_session_minutes": 102.0,
    "longest_session_minutes": 180,
    "shortest_session_minutes": 30
  },
  "quality": {
    "resolutions": {
      "1080p": 20,
      "720p": 5
    },
    "framerates": {
      "30": 18,
      "60": 7
    },
    "average_bitrate_mbps": 5.2
  },
  "uploads": {
    "total_uploaded": 25,
    "failed_uploads": 0,
    "average_upload_speed_mbps": 2.5,
    "total_upload_time_minutes": 125.0
  }
}
```

## Benefits of This Structure

1. **Easy Date Navigation**: Find all recordings from a specific date quickly
2. **Organized by Streamer**: Each streamer has their own directory
3. **Scalable**: Can handle thousands of recordings without performance issues
4. **Daily Summaries**: Quick overview of each day's recordings
5. **Historical Analysis**: Easy to analyze recording patterns over time
6. **Backup Friendly**: Can backup individual dates or streamers

## Usage

### Python Database Manager

```bash
# Initialize a new streamer
python scripts/database_manager.py init honeyyykate chaturbate

# View overall statistics for a streamer
python scripts/database_manager.py stats honeyyykate

# View recordings for a specific date
python scripts/database_manager.py date honeyyykate 2026-04-19

# List all dates with recordings for a streamer
python scripts/database_manager.py dates honeyyykate

# List all streamers
python scripts/database_manager.py list

# Export global statistics
python scripts/database_manager.py export

# Create a full backup
python scripts/database_manager.py backup
```

### GitHub Actions Workflow

The workflow automatically:
1. Records streams for each channel
2. Converts to MP4 and uploads to GoFile
3. Saves to `database/<username>/<date>/recordings.json`
4. Updates `database/<username>/stats.json`
5. Commits changes back to the repository

### Adding a Recording Manually

```python
from scripts.database_manager import DatabaseManager

db = DatabaseManager()

record = {
    "id": "honeyyykate_1713556800_12345",
    "username": "honeyyykate",
    "site": "chaturbate",
    "filename": "honeyyykate_2026-04-19_20-00-00.mp4",
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

db.add_recording("honeyyykate", "chaturbate", record, "2026-04-19")
```

## Querying the Database

### Find all recordings from a specific month
```bash
# List all dates in April 2026
ls database/honeyyykate/ | grep "2026-04"
```

### Get total size for a specific date
```bash
jq '.summary.total_size_mb' database/honeyyykate/2026-04-19/recordings.json
```

### List all GoFile links for a streamer
```bash
find database/honeyyykate -name "recordings.json" -exec jq -r '.recordings[].gofile_link' {} \;
```

### Count recordings per day
```bash
for dir in database/honeyyykate/*/; do
  date=$(basename "$dir")
  count=$(jq '.summary.total_recordings' "$dir/recordings.json")
  echo "$date: $count recordings"
done
```

## Backup Strategy

1. **Automatic Backups**: GitHub Actions commits database changes after each run
2. **Date-Level Backups**: Each date is a separate file, easy to backup individually
3. **Full Backups**: Use `python scripts/database_manager.py backup` for complete backup
4. **Git History**: Full history available in git commits

## Maintenance

- Database files are automatically organized by date
- Statistics are recalculated after each new recording
- Old records are never deleted (append-only)
- Each date directory is independent and can be archived separately

## Migration from Old Structure

If you have an old `database/<username>/uploads.json` file, you can migrate it:

```python
from scripts.database_manager import DatabaseManager
import json
from datetime import datetime

db = DatabaseManager()

# Load old format
with open('database/honeyyykate/uploads.json', 'r') as f:
    old_data = json.load(f)

# Migrate each record
for record in old_data['records']:
    # Extract date from uploaded_at
    dt = datetime.fromisoformat(record['uploaded_at'].replace('Z', '+00:00'))
    date = dt.strftime("%Y-%m-%d")
    
    # Add to new structure
    db.add_recording(
        username=record['username'],
        site=record['site'],
        record=record,
        date=date
    )

print("Migration complete!")
```
