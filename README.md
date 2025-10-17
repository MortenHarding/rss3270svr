The code in this repo is a fork of https://github.com/ErnieTech101/rss3270svr
with the below improvements and added features.

# A 3270 RSS Server written in Go
This is a minimal example of a **3270 (TN3270)** terminal server in Go that displays an RSS feeds on a 24×80 style “green screen” using the `racingmars/go3270` library.

---
## Features

- Connect via a 3270 emulator (e.g. `wx3270`, `wc3270`, Vista or Mocha for Mac) to port **7300**  
- Displays top headlines from a selected RSS feed  
- Switch between different RSS feeds
- Add a custom RSS feed
- Customize the list of RSS feeds presented using the file `rssfeed.url`
- First row in `rssfeed.url` is the default RSS feed
- Handle some special characters, not in EBCDIC. Currently only Nordic characters.
- Refresh the RSS feed when you press **Enter**
- Select another RSS feed by pressing **PF4**
- Type `q` + Enter to quit, or press **PF3** / **Clear**      

---
## Requirements

- Network access from client to rss3270svr on port 7300, which is the default, or set port using the command line parameter -port xxxx
- The file [rssfeed.url](https://github.com/MortenHarding/rss3270svr/blob/main/rssfeed.url)
- A TN3270 emulator on client side (e.g. x3270, wc3270)

---
## How to use it

Get the latest releae of [rss3270svr](https://github.com/MortenHarding/rss3270svr/releases) from github, and the file [rssfeed.url](https://github.com/MortenHarding/rss3270svr/blob/main/rssfeed.url). Place both files in the same directory, and start rss3270svr.

 `./rss3270svr`

The default port is 7300 that you will access from your TN3270 terminal emulator.
Select your own port, using the command line parameter -port

 `./rss3270svr -port 9010`

---
## How to connect
Connect to the server's IP with a 3270 Client using port 7300 and a model 2 terminal style

Example: `x3270 localhost:7300`

---
## Compile your own rss3270svr executable

 `git clone https://github.com/MortenHarding/rss3270svr.git`

 `cd rss3270svr`
 
 `go mod init rss3270svr`

Add the githut racingmars Go3270 dependency:
   
 `go get github.com/racingmars/go3270@latest`
 
 `go mod tidy`

Build an executable

 `go build -o rss3270svr rss3270svr.go`
 

---
## License / Attribution
This code is free to use, experiment with, and modify.

The 3270 handling logic uses racingmars/go3270 (MIT-style / open source) as the backend for TN3270 screens.
