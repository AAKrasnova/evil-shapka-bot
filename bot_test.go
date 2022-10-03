package main

import "testing"

func Test_trimMeta(t *testing.T) {
	type args struct {
		name []string
		text string
	}
	tests := []struct {
		name       string
		args       args
		wantResult string
	}{
		{
			name:       "case 1",
			args:       args{name: names.WordCount, text: "Слов: 2345"},
			wantResult: "2345",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotResult := trimMeta(tt.args.name, tt.args.text); gotResult != tt.wantResult {
				t.Errorf("trimMeta() = %v, want %v", gotResult, tt.wantResult)
			}
		})
	}
}
