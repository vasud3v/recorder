#!/usr/bin/env python3
"""
Database Manager for GoondVR
Manages recording metadata organized by streamer/date
"""

import os
import json
import sys
from pathlib import Path
from datetime import datetime
from typing import Dict, List, Optional


class DatabaseManager:
    """Manage GoondVR database structure: database/<streamer>/<date>/recordings.json"""
    
    def __init__(self, base_dir="database"):
        self.base_dir = Path(base_dir)
        self.base_dir.mkdir(exist_ok=True)
        
        # Create subdirectories
        (self.base_dir / "backups").mkdir(exist_ok=True)
        (self.base_dir / "global").mkdir(exist_ok=True)
    
    def get_streamer_dir(self, username: str) -> Path:
        """Get or create streamer directory"""
        streamer_dir = self.base_dir / username
        streamer_dir.mkdir(exist_ok=True)
        return streamer_dir
    
    def get_date_dir(self, username: str, date: str = None) -> Path:
        """Get or create date directory for a streamer (YYYY-MM-DD format)"""
        if date is None:
            date = datetime.now().strftime("%Y-%m-%d")
        
        date_dir = self.get_streamer_dir(username) / date
        date_dir.mkdir(exist_ok=True)
        return date_dir
    
    def init_streamer(self, username: str, site: str = "chaturbate") -> Dict:
        """Initialize a new streamer in the database"""
        streamer_dir = self.get_streamer_dir(username)
        
        # Initialize stats.json for the streamer
        stats_file = streamer_dir / "stats.json"
        if not stats_file.exists():
            stats_data = {
                "username": username,
                "site": site,
                "created_at": datetime.now().isoformat(),
                "last_updated": datetime.now().isoformat(),
                "statistics": {
                    "total_recordings": 0,
                    "total_size_gb": 0.0,
                    "total_duration_hours": 0.0,
                    "total_days_recorded": 0,
                    "first_recording_date": None,
                    "last_recording_date": None,
                    "average_recordings_per_day": 0.0,
                    "average_session_minutes": 0.0,
                    "longest_session_minutes": 0,
                    "shortest_session_minutes": 0
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
        
        return {"stats": str(stats_file)}
    
    def init_date(self, username: str, date: str = None) -> Dict:
        """Initialize a date directory for recordings"""
        date_dir = self.get_date_dir(username, date)
        
        # Initialize recordings.json for this date
        recordings_file = date_dir / "recordings.json"
        if not recordings_file.exists():
            recordings_data = {
                "date": date or datetime.now().strftime("%Y-%m-%d"),
                "username": username,
                "recordings": [],
                "summary": {
                    "total_recordings": 0,
                    "total_size_bytes": 0,
                    "total_size_mb": 0.0,
                    "total_duration_seconds": 0,
                    "total_duration_minutes": 0.0,
                    "first_recording_time": None,
                    "last_recording_time": None
                }
            }
            with open(recordings_file, 'w') as f:
                json.dump(recordings_data, f, indent=2)
        
        # Initialize metadata.json for this date
        metadata_file = date_dir / "metadata.json"
        if not metadata_file.exists():
            metadata_data = {
                "date": date or datetime.now().strftime("%Y-%m-%d"),
                "username": username,
                "created_at": datetime.now().isoformat(),
                "notes": "",
                "tags": []
            }
            with open(metadata_file, 'w') as f:
                json.dump(metadata_data, f, indent=2)
        
        return {
            "recordings": str(recordings_file),
            "metadata": str(metadata_file)
        }
    
    def add_recording(self, username: str, site: str, record: Dict, date: str = None) -> bool:
        """Add a recording to the database"""
        if date is None:
            # Extract date from uploaded_at timestamp
            try:
                dt = datetime.fromisoformat(record["uploaded_at"].replace('Z', '+00:00'))
                date = dt.strftime("%Y-%m-%d")
            except:
                date = datetime.now().strftime("%Y-%m-%d")
        
        # Initialize streamer if needed
        self.init_streamer(username, site)
        
        # Initialize date directory if needed
        self.init_date(username, date)
        
        # Add recording to date file
        date_dir = self.get_date_dir(username, date)
        recordings_file = date_dir / "recordings.json"
        
        with open(recordings_file, 'r') as f:
            data = json.load(f)
        
        # Add record
        data["recordings"].append(record)
        
        # Update summary
        total_size = sum(r["filesize_bytes"] for r in data["recordings"])
        total_duration = sum(r.get("duration_seconds", 0) for r in data["recordings"])
        count = len(data["recordings"])
        
        times = [r["uploaded_at"] for r in data["recordings"]]
        
        data["summary"] = {
            "total_recordings": count,
            "total_size_bytes": total_size,
            "total_size_mb": round(total_size / (1024**2), 2),
            "total_duration_seconds": total_duration,
            "total_duration_minutes": round(total_duration / 60, 2),
            "first_recording_time": min(times) if times else None,
            "last_recording_time": max(times) if times else None
        }
        
        # Save
        with open(recordings_file, 'w') as f:
            json.dump(data, f, indent=2)
        
        # Update streamer stats
        self.update_streamer_stats(username, site)
        
        return True
    
    def update_streamer_stats(self, username: str, site: str) -> bool:
        """Recalculate overall statistics for a streamer"""
        streamer_dir = self.get_streamer_dir(username)
        stats_file = streamer_dir / "stats.json"
        
        # Collect all recordings from all dates
        all_recordings = []
        all_dates = []
        
        for date_dir in sorted(streamer_dir.iterdir()):
            if date_dir.is_dir() and date_dir.name not in ["backups", "global"]:
                recordings_file = date_dir / "recordings.json"
                if recordings_file.exists():
                    with open(recordings_file, 'r') as f:
                        date_data = json.load(f)
                    all_recordings.extend(date_data["recordings"])
                    all_dates.append(date_dir.name)
        
        if not all_recordings:
            return False
        
        # Calculate statistics
        durations = [r.get("duration_seconds", 0) / 60 for r in all_recordings]  # minutes
        sizes = [r["filesize_bytes"] for r in all_recordings]
        upload_times = [r.get("upload_duration", 0) for r in all_recordings]
        
        # Resolution and framerate distribution
        resolutions = {}
        framerates = {}
        for r in all_recordings:
            res = r.get("resolution", "unknown")
            fps = r.get("framerate", 0)
            resolutions[res] = resolutions.get(res, 0) + 1
            framerates[str(fps)] = framerates.get(str(fps), 0) + 1
        
        # Date range
        first_date = min(all_dates) if all_dates else None
        last_date = max(all_dates) if all_dates else None
        total_days = len(all_dates)
        
        stats_data = {
            "username": username,
            "site": site,
            "created_at": datetime.now().isoformat(),
            "last_updated": datetime.now().isoformat(),
            "statistics": {
                "total_recordings": len(all_recordings),
                "total_size_gb": round(sum(sizes) / (1024**3), 2),
                "total_duration_hours": round(sum(durations) / 60, 2),
                "total_days_recorded": total_days,
                "first_recording_date": first_date,
                "last_recording_date": last_date,
                "average_recordings_per_day": round(len(all_recordings) / total_days, 2) if total_days > 0 else 0.0,
                "average_session_minutes": round(sum(durations) / len(durations), 2) if durations else 0.0,
                "longest_session_minutes": round(max(durations), 2) if durations else 0,
                "shortest_session_minutes": round(min(durations), 2) if durations else 0
            },
            "quality": {
                "resolutions": resolutions,
                "framerates": framerates,
                "average_bitrate_mbps": 0.0
            },
            "uploads": {
                "total_uploaded": len(all_recordings),
                "failed_uploads": 0,
                "average_upload_speed_mbps": round(sum(r.get("upload_speed", 0) for r in all_recordings) / len(all_recordings), 2) if all_recordings else 0.0,
                "total_upload_time_minutes": round(sum(upload_times) / 60, 2)
            }
        }
        
        # Save stats
        with open(stats_file, 'w') as f:
            json.dump(stats_data, f, indent=2)
        
        return True
    
    def get_streamer_stats(self, username: str) -> Optional[Dict]:
        """Get overall statistics for a streamer"""
        streamer_dir = self.get_streamer_dir(username)
        stats_file = streamer_dir / "stats.json"
        
        if not stats_file.exists():
            return None
        
        with open(stats_file, 'r') as f:
            return json.load(f)
    
    def get_date_recordings(self, username: str, date: str) -> Optional[Dict]:
        """Get all recordings for a specific date"""
        date_dir = self.get_date_dir(username, date)
        recordings_file = date_dir / "recordings.json"
        
        if not recordings_file.exists():
            return None
        
        with open(recordings_file, 'r') as f:
            return json.load(f)
    
    def list_streamers(self) -> List[str]:
        """List all streamers in the database"""
        streamers = []
        
        for streamer_dir in self.base_dir.iterdir():
            if streamer_dir.is_dir() and streamer_dir.name not in ["backups", "global"]:
                streamers.append(streamer_dir.name)
        
        return sorted(streamers)
    
    def list_dates(self, username: str) -> List[str]:
        """List all dates with recordings for a streamer"""
        streamer_dir = self.get_streamer_dir(username)
        dates = []
        
        for date_dir in streamer_dir.iterdir():
            if date_dir.is_dir() and date_dir.name not in ["backups", "global"]:
                dates.append(date_dir.name)
        
        return sorted(dates)
    
    def export_global_database(self) -> Dict:
        """Export combined database for all streamers"""
        global_data = {
            "exported_at": datetime.now().isoformat(),
            "streamers": {},
            "global_statistics": {
                "total_streamers": 0,
                "total_recordings": 0,
                "total_size_gb": 0.0,
                "total_duration_hours": 0.0,
                "total_days_recorded": 0
            }
        }
        
        total_recordings = 0
        total_size = 0
        total_duration = 0
        total_days = 0
        
        for streamer in self.list_streamers():
            stats = self.get_streamer_stats(streamer)
            if stats:
                global_data["streamers"][streamer] = stats
                total_recordings += stats["statistics"]["total_recordings"]
                total_size += stats["statistics"]["total_size_gb"] * (1024**3)
                total_duration += stats["statistics"]["total_duration_hours"] * 60
                total_days += stats["statistics"]["total_days_recorded"]
        
        global_data["global_statistics"] = {
            "total_streamers": len(global_data["streamers"]),
            "total_recordings": total_recordings,
            "total_size_gb": round(total_size / (1024**3), 2),
            "total_duration_hours": round(total_duration / 60, 2),
            "total_days_recorded": total_days
        }
        
        # Save to global directory
        global_file = self.base_dir / "global" / "all_stats.json"
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
        print("  python database_manager.py init <username> [site]")
        print("  python database_manager.py stats <username>")
        print("  python database_manager.py date <username> <YYYY-MM-DD>")
        print("  python database_manager.py list")
        print("  python database_manager.py dates <username>")
        print("  python database_manager.py export")
        print("  python database_manager.py backup")
        print("\nExamples:")
        print("  python database_manager.py init honeyyykate chaturbate")
        print("  python database_manager.py stats honeyyykate")
        print("  python database_manager.py date honeyyykate 2026-04-19")
        print("  python database_manager.py dates honeyyykate")
        print("  python database_manager.py list")
        sys.exit(1)
    
    db = DatabaseManager()
    command = sys.argv[1]
    
    if command == "init":
        if len(sys.argv) < 3:
            print("Error: username required")
            sys.exit(1)
        
        username = sys.argv[2]
        site = sys.argv[3] if len(sys.argv) > 3 else "chaturbate"
        
        files = db.init_streamer(username, site)
        print(f"✓ Initialized streamer: {username}@{site}")
        print(f"  Stats: {files['stats']}")
    
    elif command == "stats":
        if len(sys.argv) < 3:
            print("Error: username required")
            sys.exit(1)
        
        username = sys.argv[2]
        stats = db.get_streamer_stats(username)
        
        if stats:
            print(f"\n{'='*60}")
            print(f"Statistics for {username}")
            print(f"{'='*60}")
            print(f"\nRecordings: {stats['statistics']['total_recordings']}")
            print(f"Total Size: {stats['statistics']['total_size_gb']} GB")
            print(f"Total Duration: {stats['statistics']['total_duration_hours']} hours")
            print(f"Days Recorded: {stats['statistics']['total_days_recorded']}")
            print(f"Average/Day: {stats['statistics']['average_recordings_per_day']} recordings")
            print(f"Average Session: {stats['statistics']['average_session_minutes']} minutes")
            print(f"\nFirst Recording: {stats['statistics']['first_recording_date']}")
            print(f"Last Recording: {stats['statistics']['last_recording_date']}")
            print(f"\nResolutions: {stats['quality']['resolutions']}")
            print(f"Framerates: {stats['quality']['framerates']}")
        else:
            print(f"No stats found for {username}")
    
    elif command == "date":
        if len(sys.argv) < 4:
            print("Error: username and date required")
            sys.exit(1)
        
        username = sys.argv[2]
        date = sys.argv[3]
        
        data = db.get_date_recordings(username, date)
        if data:
            print(f"\n{'='*60}")
            print(f"Recordings for {username} on {date}")
            print(f"{'='*60}")
            print(f"\nTotal Recordings: {data['summary']['total_recordings']}")
            print(f"Total Size: {data['summary']['total_size_mb']} MB")
            print(f"Total Duration: {data['summary']['total_duration_minutes']} minutes")
            print(f"\nRecordings:")
            for i, rec in enumerate(data['recordings'], 1):
                print(f"\n  [{i}] {rec['filename']}")
                print(f"      Size: {rec['filesize_bytes'] / (1024**2):.2f} MB")
                print(f"      Duration: {rec.get('duration_seconds', 0) / 60:.2f} minutes")
                print(f"      Link: {rec['gofile_link']}")
        else:
            print(f"No recordings found for {username} on {date}")
    
    elif command == "dates":
        if len(sys.argv) < 3:
            print("Error: username required")
            sys.exit(1)
        
        username = sys.argv[2]
        dates = db.list_dates(username)
        
        print(f"\n{'='*60}")
        print(f"Recording Dates for {username}")
        print(f"{'='*60}")
        for date in dates:
            data = db.get_date_recordings(username, date)
            if data:
                print(f"\n{date}: {data['summary']['total_recordings']} recordings, {data['summary']['total_size_mb']} MB")
    
    elif command == "list":
        streamers = db.list_streamers()
        print(f"\n{'='*60}")
        print(f"Streamers in Database ({len(streamers)})")
        print(f"{'='*60}")
        for streamer in streamers:
            stats = db.get_streamer_stats(streamer)
            if stats:
                print(f"\n{streamer}")
                print(f"  Recordings: {stats['statistics']['total_recordings']}")
                print(f"  Size: {stats['statistics']['total_size_gb']} GB")
                print(f"  Days: {stats['statistics']['total_days_recorded']}")
    
    elif command == "export":
        data = db.export_global_database()
        print(f"\n{'='*60}")
        print("Global Database Export")
        print(f"{'='*60}")
        print(f"\nTotal Streamers: {data['global_statistics']['total_streamers']}")
        print(f"Total Recordings: {data['global_statistics']['total_recordings']}")
        print(f"Total Size: {data['global_statistics']['total_size_gb']} GB")
        print(f"Total Duration: {data['global_statistics']['total_duration_hours']} hours")
        print(f"\nExported to: database/global/all_stats.json")
    
    elif command == "backup":
        backup_file = db.backup_database()
        print(f"✓ Database backed up to: {backup_file}")
    
    else:
        print(f"Unknown command: {command}")
        sys.exit(1)


if __name__ == "__main__":
    main()
