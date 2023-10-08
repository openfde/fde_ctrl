package fs_fusion

import "testing"

func Test_validPermR(t *testing.T) {
	type args struct {
		uid  uint32
		duid uint32
		gid  uint32
		dgid uint32
		perm uint32
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "owner ",
			args: args{
				uid:  1000,
				duid: 1000,
				gid:  1000,
				dgid: 1000,
				perm: 0b100000111111111, //o40777,16895
			},
			want: true,
		},
		{
			name: "group",
			args: args{
				uid:  1000,
				duid: 1000,
				gid:  0,
				dgid: 0,
				perm: 0b100000111100100, //40744， 16868
			},
			want: true,
		},
		{
			name: "other",
			args: args{
				uid:  1000,
				duid: 1000,
				gid:  0,
				dgid: 0,
				perm: 0b100000111100100, //40744， 16868
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validPermR(tt.args.uid, tt.args.duid, tt.args.gid, tt.args.dgid, tt.args.perm); got != tt.want {
				t.Errorf("validPermR() = %v, want %v", got, tt.want)
			}
		})
	}
}
