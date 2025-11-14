package terminal

const uosTerminalGsettingSchemas = "com.deepin.desktop.default-applications.terminal"
const uosTerminalGsetttingKey = "exec"

type UosTerminalImpl struct {
}

func (impl UosTerminalImpl) GetTerminal() (string, string) {
	// cmd := exec.Command("gsettings", "get", uosTerminalGsettingSchemas, uosTerminalGsetttingKey)
	// output, err := cmd.Output()
	// if err != nil {
	// 	logger.Error("uos_terminal_get_exec", uosTerminalGsettingSchemas, err)
	// }
	return "deepin-terminal", "/usr/bin/deepin-terminal"
}
