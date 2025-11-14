package terminal

type UbuntuTerminalImpl struct {
}

func (impl UbuntuTerminalImpl) GetTerminal() (app, path string) {
	// return "/etc/alternatives/x-terminal-emulator"
	return "gnome-terminal", "/usr/bin/gnome-terminal"
}
