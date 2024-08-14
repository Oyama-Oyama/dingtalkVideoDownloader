module DingTalk

go 1.22.6

replace DingTalk/VideoDownloader => ../video

replace DingTalk/m3u8Downloader => ../m3u8Downloader

require DingTalk/VideoDownloader v0.0.0-00010101000000-000000000000

require DingTalk/m3u8Downloader v0.0.0-00010101000000-000000000000 // indirect

require (
	github.com/chromedp/cdproto v0.0.0-20240810084448-b931b754e476 // indirect
	github.com/chromedp/chromedp v0.10.0 // indirect
	github.com/chromedp/sysutil v1.0.0 // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	golang.org/x/sys v0.22.0 // indirect
)