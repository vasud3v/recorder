# GoondVR Database Structure

## Overview

The database is now fully organized with a date-based structure for easy navigation and scalability.

## Structure

```
database/
├── README.md                          # Complete documentation
├── backups/                           # Timestamped backups
│   ├── full_backup_2026-04-19_20-15-25.json
│   ├── channels_backup_*.json
│   ├── settings_backup_*.json
│   └── uploaded_links.json           # Python upload records
├── global/                            # Global statistics
│   └── all_stats.json                # Combined stats for all streamers
└── <streamer_name>/                   # Per-streamer directory
    ├── stats.json                    # Overall streamer statistics
    ├── 2026-04-19/                   # Date-specific recordings
    │   ├── recordings.json           # All recordings from this date
    │   └── metadata.json             # Date metadata (notes, tags)
    ├── 2026-04-20/
    │   ├── recordings.json
    │   └── metadata.json
    └── ...
```

## Example: honeyyykate

```
database/honeyyykate/
├── stats.json                        # Overall stats (25 recordings, 15.5GB, 10 days)
├── 2026-04-19/
│   ├── recordings.json              # 3 recordings, 2.1GB, 180 minutes
│   └── metadata.json                # Notes: "Great stream!"
├── 2026-04-20/
│   ├── recordings.json              # 2 recordings, 1.5GB, 120 minutes
│   └── metadata.json
└── 2026-04-21/
    ├── recordings.json              # 1 recording, 800MB, 60 minutes
    └── metadata.json
```

## File Contents

### recordings.json (Per-Date)
Contains all recordings from a specific date with summary statistics.

### metadata.json (Per-Date)
Optional notes and tags for each recording day.

### stats.json (Per-Streamer)
Overall statistics across all dates:
- Total recordings, size, duration
- Days recorded
- Average recordings per day
- Quality distribution (resolutions, framerates)
- Upload statistics

## Management Commands

```bash
# Initialize a new streamer
python scripts/database_manager.py init honeyyykate chaturbate

# View overall statistics
python scripts/database_manager.py stats honeyyykate

# View recordings for a specific date
python scripts/database_manager.py date honeyyykate 2026-04-19

# List all recording dates
python scripts/database_manager.py dates honeyyykate

# List all streamers
python scripts/database_manager.py list

# Export global statistics
python scripts/database_manager.py export

# Create full backup
python scripts/database_manager.py backup
```

## Benefits

1. **Easy Navigation**: Find recordings by date instantly
2. **Scalable**: Handles thousands of recordings efficiently
3. **Daily Summaries**: Quick overview of each day
4. **Historical Analysis**: Track patterns over time
5. **Selective Backup**: Backup individual dates or streamers
6. **Git Friendly**: Small, focused commits per date

## GitHub Actions Integration

The workflow automatically:
1. Records streams
2. Uploads to GoFile
3. Saves to `database/<streamer>/<date>/recordings.json`
4. Updates `database/<streamer>/stats.json`
5. Commits changes to repository

## Querying Examples

```bash
# Find all recordings from April 2026
ls database/honeyyykate/ | grep "2026-04"

# Get total size for a specific date
jq '.summary.total_size_mb' database/honeyyykate/2026-04-19/recordings.json

# List all GoFile links
find database/honeyyykate -name "recordings.json" -exec jq -r '.recordings[].gofile_link' {} \;

# Count recordings per day
for dir in database/honeyyykate/*/; do
  date=$(basename "$dir")
  count=$(jq '.summary.total_recordings' "$dir/recordings.json" 2>/dev/null || echo "0")
  echo "$date: $count recordings"
done
```

## Migration from Old Format

If you have old `database/<username>/uploads.json` files, they will continue to work alongside the new structure. The workflow saves to both formats for backward compatibility.

## Backup Strategy

1. **Automatic**: GitHub commits after each run
2. **Date-Level**: Each date is independent
3. **Full Backup**: `python scripts/database_manager.py backup`
4. **Git History**: Complete history in commits
