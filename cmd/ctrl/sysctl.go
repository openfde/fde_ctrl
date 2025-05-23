package main

import (
	"bytes"
	"errors"
	"fde_ctrl/logger"
	"fde_ctrl/process_chan"
	"os"
	"strconv"
	"time"
)

const allowedPidMax = 65535

func checkPidMax() (shouldExit bool) {
	shouldExit = true
	//读取proc/sys/kernel/pid_max
	max, err := os.ReadFile("/proc/sys/kernel/pid_max")
	if err != nil {
		logger.Error("read_pid_max", nil, err)
		return
	}
	//判断值是否大雨65535
	//将max解析成数字
	//去掉max的换行符
	max = bytes.TrimSpace(max)
	iMax, err := strconv.Atoi(string(max))
	if err != nil {
		logger.Error("parse_pid_max", nil, err)
		return
	}

	if iMax > allowedPidMax {
		logger.Error("compare_pid_max", nil, errors.New("pid_max is too large "+string(max)))
		return
	}
	return false
}

func CheckPidMax() (shouldExit bool) {
	go detectPidMaxTimingly()
	return checkPidMax()
}

func detectPidMaxTimingly() {
	defer func() {
		if err := recover(); err != nil {
			logger.Error("recover_pid_max_timingly", nil, nil)
		}
	}()
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if checkPidMax() {
				logger.Error("pid_max_check", nil, errors.New("pid_max check failed"))
				process_chan.SendUnexpected()
			}
		}
	}
}
