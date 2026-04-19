#!/usr/bin/env python3
"""
Upload videos to GoFile and store links in JSON
"""

import os
import sys
import json
import requests
from pathlib import Path
from datetime import datetime


class GoFileUploader:
    """Upload files to GoFile.io and manage links"""
    
    def __init__(self, links_file="uploaded_links.json"):
        self.links_file = links_file
        self.links_data = self.load_links()
        
    def load_links(self):
        """Load existing links from JSON file"""
        if Path(self.links_file).exists():
            with open(self.links_file, 'r') as f:
                return json.load(f)
        return {"uploads": []}
    
    def save_links(self):
        """Save links to JSON file"""
        with open(self.links_file, 'w') as f:
            json.dump(self.links_data, f, indent=2)
        print(f"✅ Links saved to {self.links_file}")
    
    def get_server(self):
        """Get the best GoFile server for upload"""
        try:
            response = requests.get("https://api.gofile.io/servers", timeout=10)
            if response.status_code == 200:
                data = response.json()
                if data['status'] == 'ok':
                    server = data['data']['servers'][0]['name']
                    print(f"📡 Using server: {server}")
                    return server
        except Exception as e:
            print(f"⚠️  Error getting server: {e}")
        
        # Fallback to default server
        return "store1"
    
    def upload_file(self, file_path):
        """Upload file to GoFile"""
        
        file_path = Path(file_path)
        
        if not file_path.exists():
            print(f"❌ File not found: {file_path}")
            return None
        
        file_size_mb = file_path.stat().st_size / (1024 * 1024)
        print(f"\n📤 Uploading: {file_path.name}")
        print(f"📊 Size: {file_size_mb:.2f} MB")
        
        # Get best server
        server = self.get_server()
        upload_url = f"https://{server}.gofile.io/contents/uploadfile"
        
        try:
            # Upload file
            with open(file_path, 'rb') as f:
                files = {'file': (file_path.name, f)}
                
                print("⏳ Uploading... (this may take a while)")
                response = requests.post(upload_url, files=files, timeout=300)
            
            if response.status_code == 200:
                data = response.json()
                
                if data['status'] == 'ok':
                    # Handle different response structures
                    file_data = data.get('data', {})
                    
                    # Try different possible field names
                    download_page = file_data.get('downloadPage') or file_data.get('link') or file_data.get('url')
                    file_id = file_data.get('id') or file_data.get('fileId') or file_data.get('code')
                    
                    if not download_page:
                        print(f"❌ Could not find download link in response")
                        return None
                    
                    print(f"\n✅ Upload successful!")
                    print(f"🔗 Download page: {download_page}")
                    if file_id:
                        print(f"🆔 File ID: {file_id}")
                    
                    # Store in JSON
                    upload_info = {
                        "filename": file_path.name,
                        "file_path": str(file_path),
                        "size_mb": round(file_size_mb, 2),
                        "download_page": download_page,
                        "file_id": file_id or "N/A",
                        "uploaded_at": datetime.now().isoformat(),
                        "server": server
                    }
                    
                    self.links_data["uploads"].append(upload_info)
                    self.save_links()
                    
                    return upload_info
                else:
                    print(f"❌ Upload failed: {data.get('status', 'unknown error')}")
                    return None
            else:
                print(f"❌ HTTP {response.status_code}: {response.text}")
                return None
                
        except requests.exceptions.Timeout:
            print("❌ Upload timed out (file too large or slow connection)")
            return None
        except Exception as e:
            print(f"❌ Error during upload: {e}")
            return None
    
    def list_uploads(self):
        """List all uploaded files"""
        if not self.links_data["uploads"]:
            print("No uploads found.")
            return
        
        print("\n" + "=" * 70)
        print("UPLOADED FILES")
        print("=" * 70)
        
        for i, upload in enumerate(self.links_data["uploads"], 1):
            print(f"\n[{i}] {upload['filename']}")
            print(f"    Size: {upload['size_mb']} MB")
            print(f"    Link: {upload['download_page']}")
            print(f"    Uploaded: {upload['uploaded_at']}")
    
    def search_upload(self, filename):
        """Search for an upload by filename"""
        for upload in self.links_data["uploads"]:
            if filename in upload['filename']:
                return upload
        return None


def main():
    if len(sys.argv) < 2:
        print("=" * 70)
        print("GOFILE UPLOADER")
        print("=" * 70)
        print("\nUsage:")
        print("  python upload_to_gofile.py <file_path>")
        print("  python upload_to_gofile.py --list")
        print("\nExamples:")
        print("  python upload_to_gofile.py video.mp4")
        print("  python upload_to_gofile.py goondvr/videos/completed/recording.mp4")
        print("  python upload_to_gofile.py --list")
        print("\nThe script will:")
        print("  • Upload the file to GoFile.io")
        print("  • Generate a download link")
        print("  • Save the link to uploaded_links.json")
        sys.exit(1)
    
    uploader = GoFileUploader()
    
    if sys.argv[1] == "--list":
        uploader.list_uploads()
        sys.exit(0)
    
    file_path = sys.argv[1]
    
    # Upload file
    result = uploader.upload_file(file_path)
    
    if result:
        print("\n" + "=" * 70)
        print("✅ UPLOAD COMPLETE")
        print("=" * 70)
        print(f"\n📥 Download link: {result['download_page']}")
        print(f"💾 Link saved to: {uploader.links_file}")
    else:
        print("\n❌ Upload failed")
        sys.exit(1)


if __name__ == "__main__":
    main()
