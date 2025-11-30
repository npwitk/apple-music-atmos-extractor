# Apple Music Dolby Atmos Downloader & Channel Extractor

A complete tool for downloading Dolby Atmos music from Apple Music and extracting individual 5.1 surround sound channels.

## Table of Contents
- [Prerequisites](#prerequisites)
- [Credits](#credits)
- [Setup](#setup)
- [Downloading Dolby Atmos Music](#downloading-dolby-atmos-music)
- [Extracting 5.1 Channels](#extracting-51-channels)
- [Output Format](#output-format)
- [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Required Software
- **macOS** (ARM64/Apple Silicon or Intel)
- **Docker Desktop** - [Download here](https://www.docker.com/products/docker-desktop/)
- **Homebrew** - Install: `/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"`
- **Go** - Install: `brew install go`
- **GPAC (MP4Box)** - [Download PKG](https://download.tsi.telecom-paristech.fr/gpac/new_builds/gpac_latest_head_macos.pkg)
- **Bento4** - Install: `brew install bento4`
- **FFmpeg** - Install: `brew install ffmpeg`

### Required Keys/Tokens

#### 1. Media User Token (Required for lyrics and downloads)
1. Open [Apple Music Web Player](https://music.apple.com) in your browser
2. Log in to your Apple Music account
3. Open Developer Tools (Right-click → Inspect or `Cmd+Option+I`)
4. Go to **Application** → **Storage** → **Cookies** → `https://music.apple.com`
5. Find and copy the value of `media-user-token`

#### 2. Apple ID Credentials (Required for wrapper authentication)
- Your Apple ID email
- Your Apple ID password
- Access to 2FA device (for initial setup)

---

## Credits

This project combines work from multiple projects and contributors:

### Core Projects
- **[zhaarey/apple-music-downloader](https://github.com/zhaarey/apple-music-downloader)** - Main downloader (Go)
  - Original by [Sorrow](https://github.com/Sorrow446)
  - Modified and enhanced by [zhaarey](https://github.com/zhaarey)

- **[zhaarey/wrapper](https://github.com/zhaarey/wrapper)** - Apple Music decryption wrapper
  - By [zhaarey](https://github.com/zhaarey)

- **[WorldObservationLog/wrapper](https://github.com/WorldObservationLog/wrapper)** - ARM64 macOS-compatible wrapper build
  - ARM64 builds by [WorldObservationLog](https://github.com/WorldObservationLog)

### Additional Resources
- **[Vexcited's macOS Setup Guide](https://gist.github.com/Vexcited/ee0db9eec4533b37f67b32ffa577eae1)** - macOS ARM64 setup instructions
- **FFmpeg** - Audio channel extraction

---

## Setup

### Step 1: Clone this Repository

```bash
git clone https://github.com/npwitk/apple-music-atmos-extractor
cd apple-music-atmos-extractor
```

### Step 2: Configure the Downloader

Edit `config.yaml`:

```yaml
media-user-token: "YOUR_MEDIA_USER_TOKEN_HERE"  # Paste your token from Apple Music web
authorization-token: "your-authorization-token"  # Leave as-is (auto-retrieves)
language: ""
lrc-type: "lyrics"
embed-lrc: true
save-lrc-file: false
embed-cover: true
cover-size: 5000x5000
cover-format: jpg
atmos-save-folder: "AM-DL-Atmos downloads"  # Where Atmos files will be saved
max-memory-limit: 256
decrypt-m3u8-port: "127.0.0.1:10020"  # IMPORTANT: Set this!
get-m3u8-port: "127.0.0.1:20020"
atmos-max: 2768  # Max bitrate: 2768 or 2448
```

### Step 3: Download and Setup the Wrapper (Decryption Service)

```bash
# Download ARM64-compatible wrapper
curl -L -o Wrapper.arm64.zip https://github.com/WorldObservationLog/wrapper/releases/download/Wrapper.arm64.b375260/Wrapper.arm64.b375260.zip

# Extract
unzip Wrapper.arm64.zip -d apple-music-wrapper
cd apple-music-wrapper

# Build Docker image
docker build --tag wrapper .
```

### Step 4: Start Docker Desktop

Make sure Docker Desktop is running (check the menu bar icon).

---

## Downloading Dolby Atmos Music

### Step 1: Start the Wrapper (Terminal 1)

#### First Time Setup (with authentication):
```bash
cd apple-music-wrapper
docker run --rm -v ./rootfs/data:/app/rootfs/data -p 10020:10020 \
  -e args="-L your-apple-id@email.com:your-password -F -H 0.0.0.0" wrapper
```

**If prompted for 2FA:** Open a new terminal and run:
```bash
cd apple-music-wrapper
echo -n 123456 > ./rootfs/data/2fa.txt  # Replace 123456 with your 2FA code
```

#### Subsequent Uses (after initial authentication):
```bash
cd apple-music-wrapper
docker run --rm -v ./rootfs/data:/app/rootfs/data -p 10020:10020 \
  -e args="-H 0.0.0.0" wrapper
```

**Keep this terminal running!** The wrapper must stay active during downloads.

### Step 2: Download Dolby Atmos Album (Terminal 2)

```bash
cd apple-music-atmos-extractor
go run main.go --atmos https://music.apple.com/us/album/ALBUM_URL
```

**Examples:**
```bash
# Download a single album
go run main.go --atmos https://music.apple.com/us/album/the-life-of-a-showgirl/1833328839

# Download a specific song
go run main.go --atmos --song https://music.apple.com/us/album/the-life-of-a-showgirl/1833328839?i=1833328840

# Download entire artist discography
go run main.go --atmos https://music.apple.com/us/artist/taylor-swift/159260351 --all-album
```

### Step 3: Wait for Download

The downloader will:
1. Fetch album metadata
2. Download album cover
3. Download each track as `.m4a` (E-AC-3 Dolby Atmos)
4. Embed metadata and artwork

**Output:** `AM-DL-Atmos downloads/Artist Name/Album Name/*.m4a`

---

## Extracting 5.1 Channels

After downloading, you can extract the individual 5.1 surround sound channels from each Atmos track.

### Step 1: Navigate to Album Folder

```bash
cd "AM-DL-Atmos downloads/Artist Name/Album Name"
```

### Step 2: Create Extraction Script

Create a file named `extract_atmos_stems.sh`:

```bash
#!/bin/bash

# Script to extract 5.1 channels from Dolby Atmos tracks
# Output: FL (Front Left), FR (Front Right), FC (Front Center),
#         LFE (Low Frequency Effects), SL (Surround Left), SR (Surround Right)

STEMS_DIR="Atmos Stems"

# Find all .m4a files
find . -maxdepth 1 -name "*.m4a" -type f | sort | while read -r file; do
    # Get the base filename without extension
    basename="${file%.m4a}"
    basename="${basename#./}"

    echo "Processing: $basename"

    # Create directory for this track's stems
    track_dir="$STEMS_DIR/$basename"
    mkdir -p "$track_dir"

    # Extract each channel individually
    echo "  Extracting Front Left..."
    ffmpeg -y -i "$file" -loglevel error -map 0:a:0 -af "pan=mono|c0=c0" -c:a pcm_s24le "$track_dir/FL-FrontLeft.wav"

    echo "  Extracting Front Right..."
    ffmpeg -y -i "$file" -loglevel error -map 0:a:0 -af "pan=mono|c0=c1" -c:a pcm_s24le "$track_dir/FR-FrontRight.wav"

    echo "  Extracting Front Center..."
    ffmpeg -y -i "$file" -loglevel error -map 0:a:0 -af "pan=mono|c0=c2" -c:a pcm_s24le "$track_dir/FC-FrontCenter.wav"

    echo "  Extracting LFE..."
    ffmpeg -y -i "$file" -loglevel error -map 0:a:0 -af "pan=mono|c0=c3" -c:a pcm_s24le "$track_dir/LFE-LowFrequency.wav"

    echo "  Extracting Surround Left..."
    ffmpeg -y -i "$file" -loglevel error -map 0:a:0 -af "pan=mono|c0=c4" -c:a pcm_s24le "$track_dir/SL-SurroundLeft.wav"

    echo "  Extracting Surround Right..."
    ffmpeg -y -i "$file" -loglevel error -map 0:a:0 -af "pan=mono|c0=c5" -c:a pcm_s24le "$track_dir/SR-SurroundRight.wav"

    echo "Completed: $basename"
    echo ""
done

echo ""
echo "========================================="
echo "All tracks processed successfully!"
echo "Stems saved to: $STEMS_DIR/"
echo "========================================="
```

### Step 3: Run the Extraction

```bash
chmod +x extract_atmos_stems.sh
./extract_atmos_stems.sh
```

### Step 4: Access Extracted Channels

```bash
cd "Atmos Stems"
ls
```

---

## Output Format

### Downloaded Atmos Files (.m4a)
- **Codec:** E-AC-3 (Enhanced AC-3) with Dolby Atmos
- **Channels:** 5.1 surround (encoded)
- **Bitrate:** Up to 768 kbps
- **Metadata:** Embedded album art, lyrics, and tags

### Extracted Channel Files (.wav)
- **Format:** 24-bit PCM WAV (lossless)
- **Channels:** 6 mono files per track
  - `FL-FrontLeft.wav` - Front Left speaker
  - `FR-FrontRight.wav` - Front Right speaker
  - `FC-FrontCenter.wav` - Front Center speaker
  - `LFE-LowFrequency.wav` - Low Frequency Effects (subwoofer)
  - `SL-SurroundLeft.wav` - Surround/Side Left speaker
  - `SR-SurroundRight.wav` - Surround/Side Right speaker
- **Sample Rate:** 48 kHz
- **Bit Depth:** 24-bit

### Directory Structure
```
AM-DL-Atmos downloads/
└── Artist Name/
    └── Album Name/
        ├── 01. Track Name.m4a
        ├── 02. Track Name.m4a
        ├── cover.jpg
        └── Atmos Stems/
            ├── 01. Track Name/
            │   ├── FL-FrontLeft.wav
            │   ├── FR-FrontRight.wav
            │   ├── FC-FrontCenter.wav
            │   ├── LFE-LowFrequency.wav
            │   ├── SL-SurroundLeft.wav
            │   └── SR-SurroundRight.wav
            └── 02. Track Name/
                └── [6 channels...]
```

---

## Troubleshooting

### "Port 10020 already in use"
- The wrapper is already running. Check if Docker container is running: `docker ps`
- Stop existing container: `docker stop $(docker ps -q --filter ancestor=wrapper)`

### "Cannot connect to Docker daemon"
- Start Docker Desktop application
- Wait until "Docker Desktop is running" appears in menu bar

### "Failed to get token"
- Ensure `media-user-token` is correctly set in `config.yaml`
- Token may have expired - get a fresh one from Apple Music web

### "Decryption failed" or "Download stuck after cover.jpg"
- Make sure the wrapper is running in Terminal 1
- Check that `decrypt-m3u8-port: "127.0.0.1:10020"` is set in `config.yaml`
- Verify Docker container is listening: `lsof -i :10020`

### "No such file or directory" when running extraction script
- Make sure you're in the album directory where the `.m4a` files are located
- Check that FFmpeg is installed: `ffmpeg -version`

### Extraction produces silent/empty WAV files
- Verify the original `.m4a` file is a valid Dolby Atmos file
- Check codec: `ffprobe file.m4a 2>&1 | grep "Audio:"`
- Should show `eac3 (Dolby Digital Plus + Dolby Atmos)`

---

## Important Notes

### What You're Getting
- The `.m4a` files contain the **complete Dolby Atmos mix** (all spatial information encoded)
- The extracted `.wav` files are the **6 base channels** of the 5.1 surround mix
- These are **NOT** individual instrument stems (vocals, drums, etc.) - those don't exist in the distributed files

### Limitations
- You cannot extract the raw Dolby Atmos objects/metadata from E-AC-3 files
- The source ADM (Audio Definition Model) files used by producers are not available to consumers
- Object-based audio information is baked into the 5.1 channel rendering

### Legal Notice
- This is for personal use and educational purposes only
- Ensure you have an active Apple Music subscription
- Respect copyright and artist rights
- Do not redistribute downloaded content

---

## Support & Contributing

### Issues
- **Downloader issues:** Report to [zhaarey/apple-music-downloader](https://github.com/zhaarey/apple-music-downloader/issues)
- **Wrapper issues:** Report to [zhaarey/wrapper](https://github.com/zhaarey/wrapper/issues)

### Contributing
Pull requests are welcome! Please ensure:
- Code follows existing style
- Include clear commit messages
- Test on macOS ARM64 and Intel if possible

---

## License

This project is provided as-is for educational purposes. Please refer to individual project licenses:
- apple-music-downloader: Check repository for license
- wrapper: Check repository for license