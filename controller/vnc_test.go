package controller

import "testing"

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
