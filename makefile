ver=`git log --pretty=format:"%h" -1`
tag=`git describe --abbrev=0 --tags`
date1=`date +%F_%T`
build:
	go build -ldflags "-X main._version_=$(ver) -X main._tag_=$(tag) -X main._date_=$(date1)"
	sudo chown root:root fde_ctrl
install:
	sudo install fde_ctrl /usr/bin/fde_ctrl -m 755
	if [ -e /usr/local/bin/mutter ]; then sudo install /usr/local/bin/mutter /usr/bin/fde_wm -m 755; else sudo install /usr/bin/mutter /usr/bin/fde_wm -m 755; fi

	if [ -e /usr/share/ukui-greeter/images/badges ];then sudo install installed/svg/fde_badge.svg /usr/share/ukui-greeter/images/badges/fde_badge.svg -m 644; fi

	sudo install conf/fde.conf /etc/fde.conf -m 644
	sudo install installed/wayland-sessions/fde.desktop /usr/share/wayland-sessions/fde.desktop -m 644 -D
	sudo install installed/lib/pm-utils/power.d/99openfde /lib/pm-utils/power.d/99openfde -m 755 
	sudo install installed/lib/systemd/system-sleep/openfde /lib/systemd/system-sleep/openfde -m 755
	sudo install installed/usr/bin/fde-set-ime-engine /usr/bin/fde-set-ime-engine -m 755
	sudo install installed/usr/share/lightdm/lightdm.conf.d/96-disable-autologin-lock.conf /usr/share/lightdm/lightdm.conf.d/96-disable-autologin-lock.conf -m 644
	sudo install installed/i3/config /etc/i3/config -m 644
	sudo install installed/sysctl.conf /etc/sysctl.conf -m 644
	sudo install installed/usr/bin/fde_utils /usr/bin/fde_utils -m 755
	sudo sysctl -p

