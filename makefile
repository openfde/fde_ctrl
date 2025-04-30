all: build 

build:
	$(MAKE) -C cmd/ctrl
	$(MAKE) -C cmd/brightness
install:
	sudo chown root:root cmd/brightness/fde_brightness
	sudo chown root:root cmd/ctrl/fde_ctrl
	sudo install cmd/ctrl/fde_ctrl /usr/bin/fde_ctrl -m 755
	if [ -e /usr/local/bin/mutter ]; then sudo install /usr/local/bin/mutter /usr/bin/fde_wm -m 755; else  if [ -e /usr/bin/mutter ]; then sudo install /usr/bin/mutter /usr/bin/fde_wm -m 755; fi fi
	sudo install cmd/brightness/fde_brightness /usr/bin/fde_brightness -m 755
	sudo chmod ug+s /usr/bin/fde_brightness

	if [ -e /usr/share/ukui-greeter/images/badges ];then sudo install installed/svg/fde_badge.svg /usr/share/ukui-greeter/images/badges/fde_badge.svg -m 644; fi

	sudo install conf/fde.conf /etc/fde.conf -m 644
	sudo install installed/wayland-sessions/fde.desktop /usr/share/wayland-sessions/fde.desktop -m 644 -D
	sudo install -d /lib/pm-utils/power.d -m 755
	sudo install installed/lib/pm-utils/power.d/99openfde /lib/pm-utils/power.d/99openfde -m 755 
	sudo install installed/lib/systemd/system-sleep/openfde /lib/systemd/system-sleep/openfde -m 755
	sudo install installed/usr/bin/fde-set-ime-engine /usr/bin/fde-set-ime-engine -m 755
	if [ -e  /usr/share/lightdm/lightdm.conf.d ]; then 	sudo install installed/usr/share/lightdm/lightdm.conf.d/96-disable-autologin-lock.conf /usr/share/lightdm/lightdm.conf.d/96-disable-autologin-lock.conf -m 644 ;fi
	sudo install installed/sysctl.conf /etc/sysctl.conf -m 644
	sudo install installed/usr/bin/fde_utils /usr/bin/fde_utils -m 755
	sudo install installed/usr/bin/fde_shortcut /usr/bin/fde_shortcut -m 755
	sudo install installed/usr/bin/fde_launch /usr/bin/fde_launch -m 755
	sudo install installed/usr/bin/fde_switch_next_desktop /usr/bin/fde_switch_next_desktop -m 755
	sudo install installed/usr/share/icons/hicolor/96x96/apps/openfde.png /usr/share/icons/hicolor/96x96/apps/openfde.png -m 644 
	sudo install installed/usr/share/applications/openfde.desktop /usr/share/applications/openfde.desktop
	sudo install installed/usr/bin/fde_display_geo.py /usr/bin/fde_display_geo.py -m 755
	sudo install -d /usr/share/backgrounds -m 755
	sudo install installed/usr/share/backgrounds/openfde.png /usr/share/backgrounds/openfde.png -m 644
	sudo sysctl -p

