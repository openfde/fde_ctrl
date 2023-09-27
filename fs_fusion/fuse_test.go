package fs_fusion

import (
	"io/fs"
	"reflect"
	"testing"
	"time"
)

// type FileInfo interface {
// 	Name() string       // base name of the file
// 	Size() int64        // length in bytes for regular files; system-dependent for others
// 	Mode() FileMode     // file mode bits
// 	ModTime() time.Time // modification time
// 	IsDir() bool        // abbreviation for Mode().IsDir()
// 	Sys() any           // underlying data source (can return nil)
// }

type mockfileInfo struct {
	name string
}

func (file mockfileInfo) Name() string {
	return file.name
}

func (file mockfileInfo) Size() int64 {
	return int64(100)
}

func (file mockfileInfo) Mode() fs.FileMode {
	return 1
}
func (file mockfileInfo) ModTime() time.Time {
	return time.Unix(123898498, 0)
}

func (file mockfileInfo) IsDir() bool {
	return false
}

func (file mockfileInfo) Sys() interface{} {
	return false
}

func Test_readDevicesAndMountPoint(t *testing.T) {
	selfmounts :=
		"36 29 8:5 /home /home rw,relatime shared:8 - ext4 /dev/sda5 rw\n" +
			"35 29 8:5 / / rw,relatime shared:7 - ext4 /dev/sda5 rw\n" +
			"807 790 7:1 / /var/lib/waydroid/rootfs/vendor ro,relatime shared:446 - ext4 /dev/loop1 ro"

	type args struct {
		mounts []byte
	}

	tests := []struct {
		name string
		args args
		want map[string]volumeAndMountPoint
	}{

		{
			name: "filter the root",
			args: args{
				mounts: []byte(selfmounts),
			},
			want: map[string]volumeAndMountPoint{
				"/dev/sda5": {
					MountPoint: "/",
					MountID:    "35",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := readDevicesAndMountPoint(tt.args.mounts); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readDevicesAndMountPoint() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_readDevicesAndMountPoint1(t *testing.T) {
	multiMounts :=
		"462 29 8:5 / /dat rw,relatime shared:7 - ext4 /dev/sda5 rw\n" +
			"36 29 8:5 /home /home rw,relatime shared:8 - ext4 /dev/sda5 rw\n" +
			"37 29 8:5 /root /root rw,relatime shared:9 - ext4 /dev/sda5 rw\n" +
			"34 29 8:5 / /data rw,relatime shared:7 - ext4 /dev/sda5 rw"

	type args struct {
		mounts []byte
	}

	tests := []struct {
		name string
		args args
		want map[string]volumeAndMountPoint
	}{

		{
			name: "filter the data",
			args: args{
				mounts: []byte(multiMounts),
			},
			want: map[string]volumeAndMountPoint{
				"/dev/sda5": {
					MountPoint: "/data",
					MountID:    "34",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := readDevicesAndMountPoint(tt.args.mounts); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readDevicesAndMountPoint() = %v, want %v", got, tt.want)
			}
		})
	}
}
