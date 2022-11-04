module github.com/Script-OS/go-ttyd

go 1.18

//replace github.com/laher/mergefs => ./mergefs

require (
	github.com/azurity/go-onefile v0.0.0-20220627085546-ed66fdd30b6c
	github.com/creack/pty v1.1.18
	github.com/gorilla/websocket v1.5.0
	//github.com/laher/mergefs v0.1.1
	golang.org/x/term v0.0.0-20220722155259-a9ba230a4035
)

require golang.org/x/sys v0.0.0-20210615035016-665e8c7367d1 // indirect
