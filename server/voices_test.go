package main

import "testing"

func makeVideo(dubbing, player, number, link string) ymVideo {
	v := ymVideo{Number: flexString(number), IframeURL: link}
	v.Data.Dubbing = dubbing
	v.Data.Player = player
	return v
}

func TestGroupYummyVideos(t *testing.T) {
	videos := []ymVideo{
		makeVideo("Озвучка AniDub", "Плеер Kodik", "2", "//kodikplayer.com/season/1/abc/720p?episode=2"),
		makeVideo("Озвучка AniDub", "Плеер Kodik", "1", "//kodikplayer.com/season/1/abc/720p?episode=1"),
		makeVideo("Озвучка AniDub", "Плеер Kodik", "10", "//kodikplayer.com/season/1/abc/720p?episode=10"),
		makeVideo("Озвучка AniDub", "Плеер Kodik", "1", "//kodikplayer.com/season/1/dup/720p?episode=1"),
		makeVideo("Озвучка AniDub", "Плеер Alloha", "1", "//alloha.yani.tv/?token=1"),
		makeVideo("Субтитры", "Плеер Kodik", "1", "//kodikplayer.com/season/2/s/720p"),
		makeVideo("Озвучка Evil", "Плеер Kodik", "1", "//evil.example.com/x"),
		makeVideo("Озвучка Http", "Плеер CVH", "1", "http://ru.yummyani.me/x"),
		makeVideo("Озвучка CVH", "Плеер CVH", "1", "//ru.yummyani.me/iframeCVH.html?a=1"),
	}
	voices := groupYummyVideos(videos)
	if len(voices) != 4 {
		t.Fatalf("want 4 groups, got %d", len(voices))
	}
	first := voices[0]
	if first.ID != "ym-anidub-kodik" || first.Label != "AniDub" || first.Player != "Kodik" || first.Kind != "embed" || first.Source != "yummy" {
		t.Fatalf("first: %+v", first)
	}
	if len(first.Episodes) != 3 {
		t.Fatalf("dedupe by number failed: %d", len(first.Episodes))
	}
	order := []string{first.Episodes[0].N, first.Episodes[1].N, first.Episodes[2].N}
	if order[0] != "1" || order[1] != "2" || order[2] != "10" {
		t.Fatalf("episode order: %v", order)
	}
	if first.Episodes[0].URL != "https://kodikplayer.com/season/1/abc/720p?episode=1" {
		t.Fatalf("url: %s", first.Episodes[0].URL)
	}
	for _, voice := range voices {
		if voice.Label == "Evil" || voice.Label == "Http" {
			t.Fatalf("bad host must be dropped: %s", voice.Label)
		}
	}
}

func TestVoiceIDsUnique(t *testing.T) {
	videos := []ymVideo{
		makeVideo("Озвучка A B", "Плеер Kodik", "1", "//kodikplayer.com/1"),
		makeVideo("Озвучка A-B", "Плеер Kodik", "1", "//kodikplayer.com/2"),
	}
	voices := groupYummyVideos(videos)
	if len(voices) != 2 || voices[0].ID == voices[1].ID {
		t.Fatalf("ids: %+v", voices)
	}
}

func TestPlayerHosts(t *testing.T) {
	good := []string{"https://kodikplayer.com/a", "//alloha.yani.tv/?x=1", "https://ru.yummyani.me/x", "https://aksor.tv/x", "https://sub.aksor.tv/x"}
	bad := []string{"https://evilkodikplayer.com/a", "http://kodikplayer.com/a", "https://kodikplayer.com.evil.com/a", "javascript:alert(1)", "https://example.com"}
	for _, link := range good {
		if _, ok := cleanPlayerURL(link); !ok {
			t.Fatalf("must be allowed: %s", link)
		}
	}
	for _, link := range bad {
		if _, ok := cleanPlayerURL(link); ok {
			t.Fatalf("must be blocked: %s", link)
		}
	}
}

func TestImageURLs(t *testing.T) {
	good := map[string]string{
		"//imgproxy.yani.tv/posters/big/1.webp": "https://imgproxy.yani.tv/posters/big/1.webp",
		"/storage/releases/posters/1/a.webp":    "https://aniliberty.top/storage/releases/posters/1/a.webp",
		"https://aniliberty.top/storage/x.jpg":  "https://aniliberty.top/storage/x.jpg",
	}
	for raw, want := range good {
		got, ok := cleanImageURL(raw)
		if !ok || got != want {
			t.Fatalf("%s -> %s %v", raw, got, ok)
		}
	}
	for _, raw := range []string{"https://evil.com/a.jpg", "https://yani.tv.evil.com/a", "file:///etc/passwd", ""} {
		if _, ok := cleanImageURL(raw); ok {
			t.Fatalf("must be blocked: %s", raw)
		}
	}
}
