package controller

import (
	"testing"
)

func TestApps_Scan(t *testing.T) {
	type args struct {
		iconsPixmapsPath string
		desktopEntryPath string
		themes           []string
		sizes            []string
	}
	tests := []struct {
		name    string
		impls   Apps
		args    args
		output  Apps
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				iconsPixmapsPath: "./test/pixmaps",
				desktopEntryPath: "./test/applications",
				themes:           []string{"hicolor"},
				sizes:            []string{"64x64"},
			},
			impls: Apps{},
			output: Apps{
				{
					Type:     "Application",
					Path:     "/usr/share/code/code --unity-launch %F",
					Icon:     "",
					IconPath: "./test/pixmaps/vscode.png",
					IconType: ".png",
					Name:     "Visual Studio Code",
					ZhName:   "VS 开发",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.impls.scan(tt.args.iconsPixmapsPath, tt.args.desktopEntryPath, tt.args.themes, tt.args.sizes)
			if (err != nil) != tt.wantErr {
				t.Errorf("Apps.AppScan() error = %v, wantErr %v", err, tt.wantErr)
			}
			for _, value := range tt.impls {
				if value.Name == "Visual Studio Code" {
					if value.IconType != tt.output[0].IconType {
						t.Errorf("Apps.AppScan() IconType %s output Icontype %s ", value.IconType, tt.output[0].IconType)
					}
					if value.Path != tt.output[0].Path {
						t.Errorf("Apps.AppScan() path %s output path %s ", value.Path, tt.output[0].Path)
					}
					if value.ZhName != tt.output[0].ZhName {
						t.Errorf("Apps.AppScan() zhname %s output zhname %s ", value.ZhName, tt.output[0].ZhName)
					}
				}
			}
		})
	}
}

func Test_validatePage(t *testing.T) {
	type args struct {
		start  int
		end    int
		length int
	}
	tests := []struct {
		name  string
		args  args
		want  int
		want1 int
	}{
		{
			name: "start great than length",
			args: args{
				start:  10,
				length: 9,
			},
			want:  9,
			want1: 9,
		},
		{
			name: "end great than length",
			args: args{
				start:  1,
				length: 9,
				end:    12,
			},
			want:  1,
			want1: 9,
		},
		{
			name: "start great than end",
			args: args{
				start:  3,
				length: 9,
				end:    1,
			},
			want:  3,
			want1: 9,
		},
		{
			name: "normal",
			args: args{
				start:  0,
				length: 9,
				end:    5,
			},
			want:  0,
			want1: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var src = [10]string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}
			got, got1 := validatePage(tt.args.start, tt.args.end, tt.args.length)
			if got != tt.want {
				t.Errorf("validatePage() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("validatePage() got1 = %v, want %v", got1, tt.want1)
			}
			t.Log(src[got:got1])
		})
	}
}
