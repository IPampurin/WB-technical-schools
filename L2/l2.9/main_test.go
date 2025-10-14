package main

import "testing"

func TestUnpackingString(t *testing.T) {

	type args struct {
		str string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"1", args{"a4bc2d5e"}, "aaaabccddddde", false},
		{"2", args{"abcd"}, "abcd", false},
		{"3", args{"45"}, "", true},
		{"4", args{""}, "", false},
		{"5", args{"qwe\\4\\5"}, "qwe45", false},
		{"6", args{"qwe\\45"}, "qwe44444", false},
		{"7", args{"\\45"}, "44444", false},
		{"8", args{"\\"}, "", true},
		{"9", args{"abc\\"}, "", true},
		{"10", args{"\\\\a"}, "\\a", false},
		{"11", args{"\\510"}, "5555555555", false},
		{"12", args{"a0"}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := unpackingString(tt.args.str)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnpackingString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("UnpackingString() = %v, want %v", got, tt.want)
			}
		})
	}
}
