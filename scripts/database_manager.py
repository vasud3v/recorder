#!/usr/bin/env python3
"""
Database Manager for GoondVR
Manages recording metadata, statistics, and upload history
"""

import os
import json
import sys
from pathlib import Path
from datetime import datetime
from typing import Dict, List, Optional


class DatabaseManager:
    """Manage GoondVR database structure"""
    
    def __init__(self, base_dir="database"):
        self.base_dir = Path(base_dir)
        self.base_dir.mkdir(exist_ok=True)
        
        # Create subdirectories
        (self.base_dir / "backups").mkdir(exist_ok=True)
        (self.base_dir / "global").mkdir(exist_ok=True)
    
    def get_channel_dir(self, username: str, site: str = "chaturbate") -> Path:
        """Get or create channel directory"""
        channel_dir = self.base_dir / username
        channel_dir.mkdir(exist_ok=True)
        return channel_dir
    
    def init_channel(self, username: str, site: str = "chaturbate", 
                     resolution: int = 1080, framerate: int = 30) -> Dict:
        """Initialize a new channel in the database"""
        channel_dir = self.get_channel_dir(username, site)
        
        # Initialize uploads.json
        uploads_file = channel_dir / "uploads.json"
        if not uploads_file.exists():
            uploads_data = {
                "channel": {
                    "username": username,
                    "site": site,
                    "first_recorded": None,
                    "last_recorded": None
                },
                "records": [],
                "summary": {
                    "total_recordings": 0,
                    "total_size_bytes": 0,
                    "total_size_gb": 0.0,
                    "total_duration_seconds": 0,
                    "total_duration_hours": 0.0,
                    "average_filesize_mb": 0.0,
                    "average_duration_minutes": 0.0
                }
            }
            with open(uploads_file, 'w') as f:
                json.dump(uploads_data, f, indent=2)
        
        # Initialize stats.json
        stats_file = channel_dir / "stats.json"
        if not stats_file.exists():
            stats_data = {
                "username": username,
                "site": site,
                "statistics": {
                    "total_recordings": 0,
                    "total_size_gb": 0.0,
                    "total_duration_hours": 0.0,
                    "first_recording": None,
                    "last_recording": None,
                    "average_session_minutes": 0.0,
                    "longest_session_minutes": 0,
                    "shortest_session_minutes": 0,
                    "recordings_per_day": 0.0,
                    "offline_streak": 0,
                    "last_online": None
                },
                "quality": {
                    "resolutions": {},
                    "framerates": {},
                    "average_bitrate_mbps": 0.0
                },
                "uploads": {
                    "total_uploaded": 0,
                    "failed_uploads": 0,
                    "average_upload_speed_mbps": 0.0,
                    "total_upload_time_minutes": 0.0
                }
            }
            with open(stats_file, 'w') as f:
                json.dump(stats_data, f, indent=2)
        
        # Initialize metadata.json
        metadata_file = channel_dir / "metadata.json"
        if not metadata_file.exists():
            metadata_data = {
                "username": username,
                "site": site,
                "display_name": username.capitalize(),
                "profile_url": f"https://{site}.com/{username}/",
                "added_at": datetime.now().isoformat(),
                "last_updated": datetime.now().isoformat(),
                "settings": {
                    "resolution": resolution,
                    "framerate": framerate,
                    "max_duration": 45,
                    "interval": 1,
                    "enabled": True,
                    "paused": False
                },
                "tags": [],
                "notes": ""
            }
            with open(metadata_file, 'w') as f:
                json.dump(metadata_data, f, indent=2)
        
        return {
            "uploads": str(uploads_file),
            "stats": str(stats_file),
            "metadata": str(metadata_file)
        }
    
    def add_recording(self, username: str, site: str, record: Dict) -> bool:
        """Add a recording to the channel database"""
        channel_dir = self.get_channel_dir(username, site)
        uploads_file = channel_dir / "uploads.json"
        
        # Load existing data
        with open(uploads_file, 'r') as f:
            data = json.load(f)
        
        # Add record
        data["records"].append(record)
        
        # Update channel timestamps
        if data["channel"]["first_recorded"] is None:
            data["channel"]["first_recorded"] = record["uploaded_at"]
        data["channel"]["last_recorded"] = record["uploaded_at"]
        
        # Recalculate summary
        total_size = sum(r["filesize_bytes"] for r in data["records"])
        total_duration = sum(r.get("duration_seconds", 0) for r in data["records"])
        count = len(data["records"])
        
        data["summary"] = {
            "total_recordings": count,
            "total_size_bytes": total_size,
            "total_size_gb": round(total_size / (1024**3), 2),
            "total_duration_seconds": total_duration,
            "total_duration_hours": round(total_duration / 3600, 2),
            "average_filesize_mb": round(total_size / count / (1024**2), 2) if count > 0 else 0.0,
            "average_duration_minutes": round(total_duration / count / 60, 2) if count > 0 else 0.0
        }
        
        # Save
        with open(uploads_file, 'w') as f:
            json.dump(data, f, indent=2)
        
        # Update stats
        self.update_stats(username, site)
        
        return True
    
    def update_stats(self, username: str, site: str) -> bool:
        """Recalculate statistics for a channel"""
        channel_dir = self.get_channel_dir(username, site)
        uploads_file = channel_dir / "uploads.json"
        stats_file = channel_dir / "stats.json"
        
        # Load uploads
        with open(uploads_file, 'r') as f:
            uploads_data = json.load(f)
        
        records = uploads_data["records"]
        if not records:
            return False
        
        # Calculate statistics
        durations = [r.get("duration_seconds", 0) / 60 for r in records]  # minutes
        sizes = [r["filesize_bytes"] for r in records]
        upload_times = [r.get("upload_duration", 0) for r in records]
        
        # Resolution and framerate distribution
        resolutions = {}
        framerates = {}
        for r in records:
            res = r.get("resolution", "unknown")
            fps = r.get("framerate", 0)
            resolutions[res] = resolutions.get(res, 0) + 1
            framerates[fps] = framerates.get(fps, 0) + 1
        
        # Time range
        first_rec = min(r["uploaded_at"] for r in records)
        last_rec = max(r["uploaded_at"] for r in records)
        
        # Calculate days between first and last
        try:
            first_dt = datetime.fromisoformat(first_rec.replace('Z', '+00:00'))
            last_dt = datetime.fromisoformat(last_rec.replace('Z', '+00:00'))
            days_span = max((last_dt - first_dt).days, 1)
        except:
            days_span = 1
        
        stats_data = {
            "username": username,
            "site": site,
            "statistics": {
                "total_recordings": len(records),
                "total_size_gb": round(sum(sizes) / (1024**3), 2),
                "total_duration_hours": round(sum(durations) / 60, 2),
                "first_recording": first_rec,
                "last_recording": last_rec,
                "average_session_minutes": round(sum(durations) / len(durations), 2) if durations else 0.0,
                "longest_session_minutes": round(max(durations), 2) if durations else 0,
                "shortest_session_minutes": round(min(durations), 2) if durations else 0,
                "recordings_per_day": round(len(records) / days_span, 2),
                "offline_streak": 0,
                "last_online": last_rec
            },
            "quality": {
                "resolutions": resolutions,
                "framerates": framerates,
                "average_bitrate_mbps": 0.0
            },
            "uploads": {
                "total_uploaded": len(records),
                "failed_uploads": 0,
                "average_upload_speed_mbps": round(sum(r.get("upload_speed", 0) for r in records) / len(records), 2) if records else 0.0,
                "total_upload_time_minutes": round(sum(upload_times) / 60, 2)
            }
        }
        
        # Save stats
        with open(stats_file, 'w') as f:
            json.dump(stats_data, f, indent=2)
        
        return True
    
    def get_channel_stats(self, username: str, site: str = "chaturbate") -> Optional[Dict]:
        """Get statistics for a channel"""
        channel_dir = self.get_channel_dir(username, site)
        stats_file = channel_dir / "stats.json"
        
        if not stats_file.exists():
            return None
        
        with open(stats_file, 'r') as f:
            return json.load(f)
    
    def list_channels(self) -> List[Dict]:
        """List all channels in the database"""
        channels = []
        
        for channel_dir in self.base_dir.iterdir():
            if channel_dir.is_dir() and channel_dir.name not in ["backups", "global"]:
                metadata_file = channel_dir / "metadata.json"
                if metadata_file.exists():
                    with open(metadata_file, 'r') as f:
                        channels.append(json.load(f))
        
        return channels
    
    def export_global_database(self) -> Dict:
        """Export combined database for all channels"""
        global_data = {
            "exported_at": datetime.now().isoformat(),
            "channels": [],
            "all_uploads": [],
            "global_statistics": {
                "total_channels": 0,
                "total_recordings": 0,
                "total_size_gb": 0.0,
                "total_duration_hours": 0.0
            }
        }
        
        total_recordings = 0
        total_size = 0
        total_duration = 0
        
        for channel_dir in self.base_dir.iterdir():
            if channel_dir.is_dir() and channel_dir.name not in ["backups", "global"]:
                uploads_file = channel_dir / "uploads.json"
                if uploads_file.exists():
                    with open(uploads_file, 'r') as f:
                        uploads_data = json.load(f)
                    
                    global_data["channels"].append(uploads_data["channel"])
                    global_data["all_uploads"].extend(uploads_data["records"])
                    
                    total_recordings += uploads_data["summary"]["total_recordings"]
                    total_size += uploads_data["summary"]["total_size_bytes"]
                    total_duration += uploads_data["summary"]["total_duration_seconds"]
        
        global_data["global_statistics"] = {
            "total_channels": len(global_data["channels"]),
            "total_recordings": total_recordings,
            "total_size_gb": round(total_size / (1024**3), 2),
            "total_duration_hours": round(total_duration / 3600, 2)
        }
        
        # Save to global directory
        global_file = self.base_dir / "global" / "all_uploads.json"
        with open(global_file, 'w') as f:
            json.dump(global_data, f, indent=2)
        
        return global_data
    
    def backup_database(self) -> str:
        """Create a timestamped backup of the entire database"""
        timestamp = datetime.now().strftime("%Y-%m-%d_%H-%M-%S")
        backup_file = self.base_dir / "backups" / f"full_backup_{timestamp}.json"
        
        backup_data = self.export_global_database()
        
        with open(backup_file, 'w') as f:
            json.dump(backup_data, f, indent=2)
        
        return str(backup_file)


def main():
    """CLI interface for database management"""
    if len(sys.argv) < 2:
        print("Database Manager for GoondVR")
        print("\nUsage:")
        print("  python database_manager.py init <username> [site] [resolution] [framerate]")
        print("  python database_manager.py stats <username> [site]")
        print("  python database_manager.py list")
        print("  python database_manager.py export")
        print("  python database_manager.py backup")
        print("\nExamples:")
        print("  python database_manager.py init honeyyykate chaturbate 1080 30")
        print("  python database_manager.py stats honeyyykate")
        print("  python database_manager.py list")
        print("  python database_manager.py export")
        sys.exit(1)
    
    db = DatabaseManager()
    command = sys.argv[1]
    
    if command == "init":
        if len(sys.argv) < 3:
            print("Error: username required")
            sys.exit(1)
        
        username = sys.argv[2]
        site = sys.argv[3] if len(sys.argv) > 3 else "chaturbate"
        resolution = int(sys.argv[4]) if len(sys.argv) > 4 else 1080
        framerate = int(sys.argv[5]) if len(sys.argv) > 5 else 30
        
        files = db.init_channel(username, site, resolution, framerate)
        print(f"✓ Initialized channel: {username}@{site}")
        print(f"  Uploads: {files['uploads']}")
        print(f"  Stats: {files['stats']}")
        print(f"  Metadata: {files['metadata']}")
    
    elif command == "stats":
        if len(sys.argv) < 3:
            print("Error: username required")
            sys.exit(1)
        
        username = sys.argv[2]
        site = sys.argv[3] if len(sys.argv) > 3 else "chaturbate"
        
        stats = db.get_channel_stats(username, site)
        if stats:
            print(f"\n{'='*60}")
            print(f"Statistics for {username}@{site}")
            print(f"{'='*60}")
            print(f"\nRecordings: {stats['statistics']['total_recordings']}")
            print(f"Total Size: {stats['statistics']['total_size_gb']} GB")
            print(f"Total Duration: {stats['statistics']['total_duration_hours']} hours")
            print(f"Average Session: {stats['statistics']['average_session_minutes']} minutes")
            print(f"Recordings/Day: {stats['statistics']['recordings_per_day']}")
            print(f"\nFirst Recording: {stats['statistics']['first_recording']}")
            print(f"Last Recording: {stats['statistics']['last_recording']}")
            print(f"\nResolutions: {stats['quality']['resolutions']}")
            print(f"Framerates: {stats['quality']['framerates']}")
        else:
            print(f"No stats found for {username}@{site}")
    
    elif command == "list":
        channels = db.list_channels()
        print(f"\n{'='*60}")
        print(f"Channels in Database ({len(channels)})")
        print(f"{'='*60}")
        for ch in channels:
            print(f"\n{ch['username']}@{ch['site']}")
            print(f"  Added: {ch['added_at']}")
            print(f"  Resolution: {ch['settings']['resolution']}p @ {ch['settings']['framerate']}fps")
            print(f"  Status: {'Enabled' if ch['settings']['enabled'] else 'Disabled'}")
    
    elif command == "export":
        data = db.export_global_database()
        print(f"\n{'='*60}")
        print("Global Database Export")
        print(f"{'='*60}")
        print(f"\nTotal Channels: {data['global_statistics']['total_channels']}")
        print(f"Total Recordings: {data['global_statistics']['total_recordings']}")
        print(f"Total Size: {data['global_statistics']['total_size_gb']} GB")
        print(f"Total Duration: {data['global_statistics']['total_duration_hours']} hours")
        print(f"\nExported to: database/global/all_uploads.json")
    
    elif command == "backup":
        backup_file = db.backup_database()
        print(f"✓ Database backed up to: {backup_file}")
    
    else:
        print(f"Unknown command: {command}")
        sys.exit(1)


if __name__ == "__main__":
    main()
