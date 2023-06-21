package process_chan

type PowerAction string

const (
	Logout   = PowerAction("logout")
	Poweroff = PowerAction("poweroff")
)

var ProcessChan = make(chan PowerAction, 2)

func sendMessage(action PowerAction) {
	ProcessChan <- action
}

func SendPoweroff() {
	sendMessage(Poweroff)
}

func SendLogout() {
	sendMessage(Logout)
}
