package main

import (
	"testing"
)

func TestApps_Scan(t *testing.T) {
	type args struct {
		iconPixmapPath   string
		iconsPath        string
		desktopEntryPath string
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
				iconPixmapPath:   "./test/pixmaps",
				iconsPath:        "./test/icons",
				desktopEntryPath: "./test/applications",
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
			err := tt.impls.Scan(tt.args.iconPixmapPath, tt.args.iconsPath, tt.args.desktopEntryPath)
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
