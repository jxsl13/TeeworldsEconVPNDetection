package main

import (
	"reflect"
	"testing"
)

// needs to have a redis database running
func Test_ipsFromRange(t *testing.T) {
	type args struct {
		lowerBound string
		upperBound string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{"small range over boundary", args{"127.255.100.254", "127.255.101.2"}, []string{"127.255.100.254", "127.255.101.1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := ipsFromRange(tt.args.lowerBound, tt.args.upperBound); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ipsFromRange() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testCidrRangeEquality(t *testing.T, cidr, lower, upper string) {
	rangeIps, _ := ipsFromRange(lower, upper)
	cidrIPs, _ := ipsFromCIDR(cidr)

	if len(rangeIps) != len(cidrIPs) {
		t.Errorf("Not equal:\ncidr  :%s\nrange: %s", cidrIPs, rangeIps)
	}

	for idx, ip := range rangeIps {
		if ip != cidrIPs[idx] {
			t.Errorf("range: %s  !=  cidr: %s", ip, cidrIPs[idx])
		}
	}
}

func Test_EqualRanges(t *testing.T) {
	testCidrRangeEquality(t, "213.182.158.200/30", "213.182.158.200", "213.182.158.203")
	testCidrRangeEquality(t, "89.246.74.0/23", "89.246.74.0", "89.246.75.255")

}
