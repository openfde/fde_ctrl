package terminal

type KylinTerminalImpl struct {
}

func (impl KylinTerminalImpl) GetTerminal() (app, path string) {
	// return "/etc/alternatives/x-terminal-emulator"
	return "mate-terminal", "/usr/bin/mate-terminal"
}
