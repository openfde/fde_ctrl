all: build install

build:
	$(MAKE) -C cmd/ctrl
	sudo chown root:root cmd/ctrl/fde_ctrl
	$(MAKE) -C cmd/brightness
	sudo chown root:root cmd/brightness/fde_brightness
install:
	sudo install cmd/ctrl/fde_ctrl /usr/bin/fde_ctrl -m 755
	if [ -e /usr/local/bin/mutter ]; then sudo install /usr/local/bin/mutter /usr/bin/fde_wm -m 755; else sudo install /usr/bin/mutter /usr/bin/fde_wm -m 755; fi
	sudo install cmd/brightness/fde_brightness /usr/bin/fde_brightness -m 755
	sudo chmod ug+s /usr/bin/fde_brightness

	sudo install conf/fde.conf /etc/fde.conf -m 644
	sudo install installed/svg/fde_badge.svg /usr/share/ukui-greeter/images/badges/fde_badge.svg -m 644
	sudo install installed/wayland-sessions/fde.desktop /usr/share/wayland-sessions/fde.desktop -m 644 -D
	sudo install installed/lib/pm-utils/power.d/99openfde /lib/pm-utils/power.d/99openfde -m 755 
	sudo install installed/lib/systemd/system-sleep/openfde /lib/systemd/system-sleep/openfde -m 755
	sudo install installed/usr/bin/fde-set-ime-engine /usr/bin/fde-set-ime-engine -m 755
	sudo install installed/usr/share/lightdm/lightdm.conf.d/96-disable-autologin-lock.conf /usr/share/lightdm/lightdm.conf.d/96-disable-autologin-lock.conf -m 644
	sudo install installed/i3/config /etc/i3/config -m 644
	sudo install installed/sysctl.conf /etc/sysctl.conf -m 644
	sudo install installed/usr/bin/fde_utils /usr/bin/fde_utils -m 755
	sudo sysctl -p

