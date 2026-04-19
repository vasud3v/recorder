# Cookie Setup Guide - Fix HTTP 403 Error

## Root Cause Analysis

The HTTP 403 error occurs because **Chaturbate's CDN requires valid session cookies** to authorize access to HLS streams.

### What We Discovered:

**✅ Working Recording (April 19, 11:53 AM):**
- Cookie length: **462 characters**
- Source: User-provided `CHATURBATE_COOKIES` GitHub secret
- FlareSolverr added: `cf_clearance` + `csrftoken`
- Result: **Successful 2+ hour recording of kittengirlxo**

**❌ Failed Recording (April 19, 2:12 PM):**
- Cookie length: **0 characters** (or expired cookies)
- Source: Only FlareSolverr cookies (`cf_clearance`, `csrftoken`)
- Result: **HTTP 403 Forbidden** when accessing HLS URL

### Key Insight:
**FlareSolverr cookies alone are NOT sufficient.** You need actual Chaturbate session cookies from a logged-in browser.

---

## How to Get Fresh Cookies

### Method 1: Using Browser DevTools (Recommended)

1. **Open Chaturbate in your browser** (Chrome/Edge/Firefox)
2. **Log in to your account** (if you have one)
3. **Open Developer Tools** (F12 or Right-click → Inspect)
4. **Go to the Network tab**
5. **Refresh the page** (F5)
6. **Click on any request** to chaturbate.com
7. **Find the "Cookie" header** in Request Headers
8. **Copy the entire cookie string**

Example cookie string:
```
csrftoken=abc123...; sessionid=xyz789...; affkey=...; agreeterms=1; cf_clearance=...
```

### Method 2: Using Cookie Export Extension

1. **Install "EditThisCookie"** or "Cookie-Editor" extension
2. **Go to chaturbate.com** and log in
3. **Click the extension icon**
4. **Export cookies** as Netscape format or copy all
5. **Format as**: `name1=value1; name2=value2; ...`

---

## Update GitHub Secret

1. **Go to your GitHub repository**
2. **Settings → Secrets and variables → Actions**
3. **Find `CHATURBATE_COOKIES`** secret
4. **Click "Update"**
5. **Paste your fresh cookie string**
6. **Save**

### Important Cookies to Include:

- `csrftoken` - CSRF protection token
- `sessionid` - Your login session (if logged in)
- `cf_clearance` - Cloudflare clearance
- `affkey` - Affiliate tracking
- `agreeterms` - Terms agreement flag
- `__cfduid`, `__cf_bm` - Cloudflare bot management

---

## Testing the Fix

After updating cookies, trigger a manual workflow run:

1. **Go to Actions tab**
2. **Click "GoondVR" workflow**
3. **Click "Run workflow"**
4. **Set parameters:**
   - Duration: `1` minute
   - Debug: `true` ✅
   - Run once: `true` ✅
5. **Click "Run workflow"**

### Expected Results:

**In the logs, you should see:**
```
Cookie length: 462 | UA length: 125
[DEBUG] Found HLS URL: https://edge28-chi.live.mmcdn.com/.../llhls.m3u8
[DEBUG] master playlist response for ...
Files: 1 | Size: 15M
SUCCESS: Recorded - offline streak reset
```

**NOT:**
```
[DEBUG] HTTP 403: https://edge28-chi.live.mmcdn.com/...
Files: 0 | Size: 4.0K
WARNING: Nothing recorded - offline streak: 1
```

---

## Cookie Expiration

Chaturbate cookies typically expire after:
- **Session cookies**: When browser closes (if not logged in)
- **Persistent cookies**: 30-90 days (if logged in)

### Signs Your Cookies Expired:

1. HTTP 403 errors in workflow logs
2. "Cookie length: 0" or very short length
3. Recordings that worked before now fail
4. Age verification errors

### Solution:
**Update cookies every 30 days** or when you see 403 errors.

---

## Why FlareSolverr Alone Isn't Enough

**FlareSolverr provides:**
- `cf_clearance` - Bypasses Cloudflare challenge
- `csrftoken` - Basic CSRF token

**But Chaturbate CDN also needs:**
- `sessionid` - Proves you're logged in (for age-restricted content)
- `affkey` - Tracking/routing information
- `agreeterms` - Age verification confirmation
- Other session-specific cookies

**Without these**, the CDN returns **403 Forbidden** even though the HLS URL is valid.

---

## Troubleshooting

### Issue: Still getting 403 after updating cookies

**Check:**
1. Cookies are properly formatted (semicolon-separated)
2. No extra spaces or newlines in cookie string
3. Cookies are from the same browser session
4. You're logged in when copying cookies
5. Age verification is completed in browser

### Issue: Cookies work locally but not in GitHub Actions

**Possible causes:**
1. IP-based restrictions (GitHub Actions uses different IPs)
2. Cookies tied to specific user-agent (update `USER_AGENT` secret too)
3. Geo-restrictions (some streams are region-locked)

### Issue: Need to update cookies too frequently

**Solutions:**
1. Stay logged in to Chaturbate in your browser
2. Use "Remember me" when logging in
3. Don't clear browser cookies
4. Consider using a dedicated browser profile for this

---

## Security Notes

⚠️ **Important:**
- Cookies contain your session information
- Don't share cookies publicly
- Use GitHub Secrets (never commit cookies to code)
- Rotate cookies if compromised
- Consider using a separate Chaturbate account for automation

---

## Summary

**The working code from April 19, 11:53 AM is already correct.**

The issue is **not the code** - it's **expired or missing cookies** in the `CHATURBATE_COOKIES` GitHub secret.

**To fix:**
1. Get fresh cookies from your browser (462+ characters)
2. Update the `CHATURBATE_COOKIES` GitHub secret
3. Test with a 1-minute recording
4. Update cookies every 30 days or when you see 403 errors

The workflow will work perfectly once you provide valid session cookies! 🎯
