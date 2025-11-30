package main

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"main/internal/config"
	"main/internal/downloader"
	"main/internal/helpers"
	"main/internal/search"
	"main/utils/ampapi"
	"main/utils/structs"

	"github.com/spf13/pflag"
)

var (
	dl_atmos      bool
	dl_aac        bool
	dl_select     bool
	dl_song       bool
	artist_select bool
	debug_mode    bool
	alac_max      *int
	atmos_max     *int
	mv_max        *int
	mv_audio_type *string
	aac_type      *string
	Config        *config.Config
	counter       structs.Counter
	okDict        = make(map[string][]int)
)

func main() {
	// Load config
	var err error
	Config, err = config.Load("config.yaml")
	if err != nil {
		fmt.Printf("load Config failed: %v", err)
		return
	}

	// Get token
	token, err := ampapi.GetToken()
	if err != nil {
		if Config.AuthorizationToken != "" && Config.AuthorizationToken != "your-authorization-token" {
			token = strings.ReplaceAll(Config.AuthorizationToken, "Bearer ", "")
		} else {
			fmt.Println("Failed to get token.")
			return
		}
	}

	// Parse flags
	var search_type string
	pflag.StringVar(&search_type, "search", "", "Search for 'album', 'song', or 'artist'")
	pflag.BoolVar(&dl_atmos, "atmos", false, "Enable atmos download mode")
	pflag.BoolVar(&dl_aac, "aac", false, "Enable adm-aac download mode")
	pflag.BoolVar(&dl_select, "select", false, "Enable selective download")
	pflag.BoolVar(&dl_song, "song", false, "Enable single song download mode")
	pflag.BoolVar(&artist_select, "all-album", false, "Download all artist albums")
	pflag.BoolVar(&debug_mode, "debug", false, "Enable debug mode")
	alac_max = pflag.Int("alac-max", Config.AlacMax, "Specify the max quality for download alac")
	atmos_max = pflag.Int("atmos-max", Config.AtmosMax, "Specify the max quality for download atmos")
	aac_type = pflag.String("aac-type", Config.AacType, "Select AAC type")
	mv_audio_type = pflag.String("mv-audio-type", Config.MVAudioType, "Select MV audio type")
	mv_max = pflag.Int("mv-max", Config.MVMax, "Specify the max quality for download MV")

	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [url1 url2 ...]\n", "[main | main.exe | go run main.go]")
		fmt.Fprintf(os.Stderr, "Search Usage: %s --search [album|song|artist] [query]\n", "[main | main.exe | go run main.go]")
		fmt.Println("\nOptions:")
		pflag.PrintDefaults()
	}

	pflag.Parse()
	Config.AlacMax = *alac_max
	Config.AtmosMax = *atmos_max
	Config.AacType = *aac_type
	Config.MVAudioType = *mv_audio_type
	Config.MVMax = *mv_max

	args := pflag.Args()

	// Handle search
	if search_type != "" {
		if len(args) == 0 {
			fmt.Println("Error: --search flag requires a query.")
			pflag.Usage()
			return
		}
		selectedUrl, err := search.Handle(search_type, args, token, Config, dl_atmos, dl_aac, &dl_song, aac_type)
		if err != nil {
			fmt.Printf("\nSearch process failed: %v\n", err)
			return
		}
		if selectedUrl == "" {
			fmt.Println("\nExiting.")
			return
		}
		os.Args = []string{selectedUrl}
	} else {
		if len(args) == 0 {
			fmt.Println("No URLs provided.")
			pflag.Usage()
			return
		}
		os.Args = args
	}

	// Handle artist URL
	if strings.Contains(os.Args[0], "/artist/") {
		urlArtistName, urlArtistID, err := downloader.GetUrlArtistName(os.Args[0], token, Config)
		if err != nil {
			fmt.Println("Failed to get artistname.")
			return
		}
		Config.ArtistFolderFormat = strings.NewReplacer(
			"{UrlArtistName}", config.LimitString(urlArtistName, Config.LimitMax),
			"{ArtistId}", urlArtistID,
		).Replace(Config.ArtistFolderFormat)
		albumArgs, err := downloader.CheckArtist(os.Args[0], token, "albums", Config, artist_select)
		if err != nil {
			fmt.Println("Failed to get artist albums.")
			return
		}
		mvArgs, err := downloader.CheckArtist(os.Args[0], token, "music-videos", Config, artist_select)
		if err != nil {
			fmt.Println("Failed to get artist music-videos.")
		}
		os.Args = append(albumArgs, mvArgs...)
	}

	// Process URLs
	albumTotal := len(os.Args)
	for {
		for albumNum, urlRaw := range os.Args {
			fmt.Printf("Queue %d of %d: ", albumNum+1, albumTotal)
			var storefront, albumId string

			// Handle different URL types...
			if strings.Contains(urlRaw, "/music-video/") {
				fmt.Println("Music Video")
				if debug_mode {
					continue
				}
				counter.Total++
				if len(Config.MediaUserToken) <= 50 {
					fmt.Println(": meida-user-token is not set, skip MV dl")
					counter.Success++
					continue
				}
				if _, err := exec.LookPath("mp4decrypt"); err != nil {
					fmt.Println(": mp4decrypt is not found, skip MV dl")
					counter.Success++
					continue
				}
				mvSaveDir := strings.NewReplacer(
					"{ArtistName}", "",
					"{UrlArtistName}", "",
					"{ArtistId}", "",
				).Replace(Config.ArtistFolderFormat)
				if mvSaveDir != "" {
					mvSaveDir = filepath.Join(Config.AlacSaveFolder, helpers.SanitizeFilename(mvSaveDir))
				} else {
					mvSaveDir = Config.AlacSaveFolder
				}
				storefront, albumId = helpers.CheckURLMv(urlRaw)
				err := downloader.DownloadMusicVideo(albumId, mvSaveDir, token, storefront, Config.MediaUserToken, nil, Config)
				if err != nil {
					fmt.Println("\u26A0 Failed to dl MV:", err)
					counter.Error++
					continue
				}
				counter.Success++
				continue
			}
			if strings.Contains(urlRaw, "/song/") {
				fmt.Printf("Song->")
				storefront, songId := helpers.CheckURLSong(urlRaw)
				if storefront == "" || songId == "" {
					fmt.Println("Invalid song URL format.")
					continue
				}
				err := downloader.RipSong(songId, token, storefront, Config.MediaUserToken, Config, &counter, okDict, dl_atmos, dl_aac, dl_select, debug_mode)
				if err != nil {
					fmt.Println("Failed to rip song:", err)
				}
				continue
			}
			parse, err := url.Parse(urlRaw)
			if err != nil {
				fmt.Printf("Invalid URL: %v", err)
				continue
			}
			var urlArg_i = parse.Query().Get("i")

			if strings.Contains(urlRaw, "/album/") {
				fmt.Println("Album")
				storefront, albumId = helpers.CheckURL(urlRaw)
				err := downloader.RipAlbum(albumId, token, storefront, Config.MediaUserToken, urlArg_i, Config, &counter, okDict, dl_atmos, dl_aac, dl_select, dl_song, debug_mode)
				if err != nil {
					fmt.Println("Failed to rip album:", err)
				}
			} else if strings.Contains(urlRaw, "/playlist/") {
				fmt.Println("Playlist")
				storefront, albumId = helpers.CheckURLPlaylist(urlRaw)
				err := downloader.RipPlaylist(albumId, token, storefront, Config.MediaUserToken, Config, &counter, okDict, dl_atmos, dl_aac, dl_select, debug_mode)
				if err != nil {
					fmt.Println("Failed to rip playlist:", err)
				}
			} else if strings.Contains(urlRaw, "/station/") {
				fmt.Printf("Station")
				storefront, albumId = helpers.CheckURLStation(urlRaw)
				if len(Config.MediaUserToken) <= 50 {
					fmt.Println(": meida-user-token is not set, skip station dl")
					continue
				}
				err := downloader.RipStation(albumId, token, storefront, Config.MediaUserToken, Config, &counter, okDict, dl_atmos, dl_aac)
				if err != nil {
					fmt.Println("Failed to rip station:", err)
				}
			} else {
				fmt.Println("Invalid type")
			}
		}

		// Print summary
		fmt.Println("Summary")
		fmt.Printf("Completed: %d\n", counter.Success)
		fmt.Printf("Warnings: %d\n", counter.Unavailable+counter.NotSong)
		fmt.Printf("Errors: %d\n", counter.Error)

		if counter.Error == 0 {
			break
		}
		fmt.Println("Error detected, press Enter to try again...")
		fmt.Scanln()
		fmt.Println("Start trying again...")
		counter = structs.Counter{}
	}
}
