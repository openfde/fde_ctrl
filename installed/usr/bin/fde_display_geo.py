#!/usr/bin/python3

import gi
gi.require_version('Gtk', '3.0')
from gi.repository import Gtk, Gdk

def get_current_screen_resolution():
    # 获取默认显示
    display = Gdk.Display.get_default()
    
    # 获取当前活跃窗口
    window = display.get_default_screen().get_active_window()
    if not window:
        print("无法获取当前活动窗口")
        return None
    
    # 获取窗口所在的监视器
    monitor = display.get_monitor_at_window(window)
    
    # 获取监视器的几何信息
    geometry = monitor.get_geometry()
    
    width = geometry.width
    height = geometry.height
    scale = monitor.get_scale_factor()
    return width * scale, height * scale

if __name__ == "__main__":
    width, height = get_current_screen_resolution()
    if width and height:
        print(width,",",height)
        exit(0)
    else:
        exit(1)
