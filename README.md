# rss3270svr - a 3270 RSS Server written in Go
This is a minimal example of a **3270 (TN3270)** terminal server in Go that displays an RSS feed (BBC World) on a 24×80 style “green screen” using the `racingmars/go3270` library.

It’s intended as a Go learning project 

---
## Features - There's not too many...for now

- Connect via a 3270 emulator (e.g. `wx3270`, `wc3270`, Vista or Mocha for Mac) to port **7300**  
- Displays top headlines from the BBC World RSS feed:  
  `https://feeds.bbci.co.uk/news/world/rss.xml`  
- Refresh the RSS feed when you press **Enter**  
- Type `q` + Enter to quit, or press **PF3** / **Clear**      

---
## Requirements

- Go installed (1.18+ or whatever version you use)  
- Network access from client to your server’s port 7300. You can change the code to change the port #
- A TN3270 emulator on client side (e.g. x3270, wc3270)
- You'll get the Github racingmars go3270 library in the installation step

---
## How to use it

rss3270svr is a simple, single file Go example so it is easy to get running. I use 7300 as the Linux server port that you will access from your TN3270 terminal emulator, so use UFW as follows to open up that port

  sudo ufw allow 7300/tcp
  
  sudo UFW reload

Then create a directory to place the rss3270svr.go file into. I like to place it into my user name home under /home. You can do as you please of course

   mkdir rss3270svr

Then copy the rss3270svr.go file in the rss3270svr directory you just made. Make sure the permissions allow you to run and edit it
   
   cd rss3270svr
   go mod init rss3270svr

Add the githut racingmars Go3270 dependency:
   
   go get github.com/racingmars/go3270@latest
   go mod tidy

You can run it via Go run
   
   go run ./rss3270svr.go

or run it after you've built it into an executable

   go build -o rss3270srv rss3270srv.go
   ./rss3270svr

Connect to the server's IP with a 3270 Client using port 7300 and a model 2 terminal style

Example: x3270 your.server.ip:7300

