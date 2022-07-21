/*
Package ttyd provides the actual functionality of go-ttyd.

you can easily use this package to implement similar functionality in your own programs.

	tty := ttyd.NewTTYd(ttyd.Config{
		Gen: func() *exec.Cmd {
			// return a *exec.Cmd that you want to execute.
		},
		MaxConn: 0, // 0 means no connection limit.
	})
	http.ListenAndServe(ttyd)
*/
package ttyd
