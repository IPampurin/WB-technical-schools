package tests

import (
	"testing"
)

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
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UnpackingString(tt.args.str)
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

/*

	"a4bc2d5e",  // "aaaabccddddde"
	"abcd",      // "abcd"
	"45",        // "" + err (некорректная строка, т.к. в строке только цифры — функция должна вернуть ошибку)
	"",          // "" (пустая строка -> пустая строка)
	"qwe\\4\\5", // "qwe45" (4 и 5 не трактуются как числа, т.к. экранированы)
	"qwe\\45",   // "qwe44444" (\4 экранирует 4, поэтому распаковывается только 5)
	"\\45",      // "44444"
	"\\",        // ""

*/
