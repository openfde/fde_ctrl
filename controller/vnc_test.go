package controller

import (
	"testing"
)

func Test_simplifyPort(t *testing.T) {
	type args struct {
		port string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "should return 100",
			args: args{
				port: "6000",
			},
			want:    "100",
			wantErr: false,
		},

		{
			name: "should return 101",
			args: args{
				port: "6001",
			},
			want:    "101",
			wantErr: false,
		},
		{
			name: "should return 2",
			args: args{
				port: "5902",
			},
			want:    "2",
			wantErr: false,
		},
		{
			name: "should return 3",
			args: args{
				port: "5903",
			},
			want:    "3",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := simplifyPort(tt.args.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("simplifyPort() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("simplifyPort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_removeLinuxArgs(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name             string
		args             args
		wantFilteredPath string
	}{
		{
			name: "no linux exec args",
			args: args{
				path: "wps -u kjkj -d w3kc9 -xf",
			},
			wantFilteredPath: "wps -u kjkj -d w3kc9 -xf",
		},
		{
			name: "have linux exec args",
			args: args{
				path: "wps -u kjkj -d w3kc9 %f",
			},
			wantFilteredPath: "wps -u kjkj -d w3kc9",
		},
		{
			name: "have linux exec args without program args",
			args: args{
				path: "wps %F",
			},
			wantFilteredPath: "wps",
		},
		{
			name: "have linux exec args without program args other",
			args: args{
				path: "wps --user=%F",
			},
			wantFilteredPath: "wps",
		},
		{
			name: "no linux exec args without program args other",
			args: args{
				path: "wps F",
			},
			wantFilteredPath: "wps F",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotFilteredPath := removeDesktopArgs(tt.args.path); gotFilteredPath != tt.wantFilteredPath {
				t.Errorf("removeLinuxArgs() = %v, want %v", gotFilteredPath, tt.wantFilteredPath)
			}
		})
	}
}

func Test_parseAppPort(t *testing.T) {
	type args struct {
		line string
	}
	tests := []struct {
		name        string
		args        args
		wantAppName string
		wantPort    string
	}{
		{
			name: "terminal",
			args: args{
				line: "/usr/bin/Xtigervnc :6 -localhost=0 -desktop terminator -rfbport 5906 -SecurityTypes None -auth /home/phytium/.Xauthority -geometry 1920x1200 -depth 24",
			},
			wantAppName: "terminator",
			wantPort:    "5906",
		},
		{
			name: "no app",
			args: args{
				line: "/usr/bin/Xtigervnc",
			},
			wantAppName: "",
			wantPort:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAppName, gotPort := parseApp(tt.args.line)
			if gotAppName != tt.wantAppName {
				t.Errorf("parseAppPort() gotAppName = %v, want %v", gotAppName, tt.wantAppName)
			}
			if gotPort != tt.wantPort {
				t.Errorf("parseAppPort() gotPort = %v, want %v", gotPort, tt.wantPort)
			}
		})
	}
}
