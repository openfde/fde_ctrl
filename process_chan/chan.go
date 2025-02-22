package process_chan

type PowerAction string

const (
	Logout   = PowerAction("logout")
	Poweroff = PowerAction("poweroff")
	Unexpected = PowerAction("unexpected")
	Restart  = PowerAction("restart")
)

var ProcessChan = make(chan PowerAction, 2)

func sendMessage(action PowerAction) {
	ProcessChan <- action
}
func SendRestart() {
	sendMessage(Restart)
}

func SendPoweroff() {
	sendMessage(Poweroff)
}

func SendLogout() {
	sendMessage(Logout)
}

func SendUnexpected() {
	sendMessage(Unexpected)
}
