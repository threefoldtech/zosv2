package primitives

import (
	"testing"
	"time"

	"github.com/threefoldtech/zos/pkg/provision"
)

func TestExpired(t *testing.T) {
	type args struct {
		r *provision.Reservation
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "expired",
			args: args{&provision.Reservation{
				Created:  time.Now().Add(-time.Minute),
				Duration: time.Second,
			}},
			want: true,
		},
		{
			name: "not expired",
			args: args{&provision.Reservation{
				Created:  time.Now(),
				Duration: time.Minute,
			}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.r.Expired(); got != tt.want {
				t.Errorf("expired() = %v, want %v", got, tt.want)
			}
		})
	}
}
