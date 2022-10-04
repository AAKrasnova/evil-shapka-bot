package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
			name:       "case 1.1",
			args:       args{name: names.WordCount, text: "Слов: 2345"},
			wantResult: "2345",
		},
		{
			name:       "case 1.2",
			args:       args{name: names.WordCount, text: "Слов: 2345 слов"},
			wantResult: "2345",
		},
		{
			name:       "case 2.1",
			args:       args{name: names.Link, text: "Ссылка: https://lolesports.com/vod/108998961199305812/1/kfZngwi0r4c"},
			wantResult: "https://lolesports.com/vod/108998961199305812/1/kfZngwi0r4c",
		},
		{
			name:       "case 2.2",
			args:       args{name: names.Link, text: "https://lolesports.com/vod/108998961199305812/1/kfZngwi0r4c"},
			wantResult: "https://lolesports.com/vod/108998961199305812/1/kfZngwi0r4c",
		},
		{
			name:       "case 2.3",
			args:       args{name: names.Link, text: "https://lolesports.com/vod/108998961199305812/1/kfZngwi0r4c - Link"},
			wantResult: "https://lolesports.com/vod/108998961199305812/1/kfZngwi0r4c",
		},
		{
			name:       "case 2.4",
			args:       args{name: names.Link, text: "https://lolesports.com/vod/108998961199305812/1/kfZngwi0r4c -Link"},
			wantResult: "https://lolesports.com/vod/108998961199305812/1/kfZngwi0r4c",
		},
		{
			name:       "case 2.5",
			args:       args{name: names.Link, text: "https://lolesports.com/vod/108998961199305812/1/kfZngwi0r4c- Link"},
			wantResult: "https://lolesports.com/vod/108998961199305812/1/kfZngwi0r4c",
		},
		{
			name:       "case 2.6",
			args:       args{name: names.Link, text: "Ссылка:https://lolesports.com/vod/108998961199305812/1/kfZngwi0r4c"},
			wantResult: "https://lolesports.com/vod/108998961199305812/1/kfZngwi0r4c",
		},
		{
			name:       "case 2.7",
			args:       args{name: names.Link, text: "Ссылка :https://lolesports.com/vod/108998961199305812/1/kfZngwi0r4c"},
			wantResult: "https://lolesports.com/vod/108998961199305812/1/kfZngwi0r4c",
		},
		{
			name:       "case 2.8",
			args:       args{name: names.Link, text: "ссылка :https://lolesports.com/vod/108998961199305812/1/kfZngwi0r4c"},
			wantResult: "https://lolesports.com/vod/108998961199305812/1/kfZngwi0r4c",
		},
		{
			name:       "case 3.1",
			args:       args{name: names.Name, text: `Название "Webinar: Connecting Inputs to Outputs at Udemy by Amplitude"`},
			wantResult: `"Webinar: Connecting Inputs to Outputs at Udemy by Amplitude"`,
		},
		{
			name:       "case 3.1.1",
			args:       args{name: names.Name, text: `"Webinar: Connecting Inputs to Outputs at Udemy by Amplitude"`},
			wantResult: `"Webinar: Connecting Inputs to Outputs at Udemy by Amplitude"`,
		},
		{
			name:       "case 3.2",
			args:       args{name: names.Name, text: "Название: Webinar: Connecting Inputs to Outputs at Udemy by Amplitude"},
			wantResult: "Webinar: Connecting Inputs to Outputs at Udemy by Amplitude",
		},
		{
			name:       "case 3.3",
			args:       args{name: names.Name, text: "Name: Webinar: Connecting Inputs to Outputs at Udemy by Amplitude"},
			wantResult: "Webinar: Connecting Inputs to Outputs at Udemy by Amplitude",
		},
		{
			name:       "case 3.4",
			args:       args{name: names.Name, text: "Webinar: Connecting Inputs to Outputs at Udemy by Amplitude"},
			wantResult: "Webinar: Connecting Inputs to Outputs at Udemy by Amplitude",
		},
		{
			name:       "case 4.1",
			args:       args{name: names.Theme, text: "StartUp; A\\B Test"},
			wantResult: "StartUp; A\\B Test",
		},
		{
			name:       "case 4.2",
			args:       args{name: names.Theme, text: "Тема : StartUp, A\\B Test"},
			wantResult: "StartUp, A\\B Test",
		},
		{
			name:       "case 5.1",
			args:       args{name: names.Sphere, text: "Сфера PM"},
			wantResult: "PM",
		},
		{
			name:       "case 5.2",
			args:       args{name: names.Sphere, text: "#ProductManagement"},
			wantResult: "ProductManagement",
		},
		{
			name:       "case 6.1",
			args:       args{name: names.Duration, text: "5мин"},
			wantResult: "5",
		},
		{
			name:       "case 6.2",
			args:       args{name: names.Duration, text: "минуты10"},
			wantResult: "10",
		},
		{
			name:       "case 6.3",
			args:       args{name: names.Duration, text: "Duration пять минут"},
			wantResult: "пять",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult := trimMeta(tt.args.name, tt.args.text)
			require.Equal(t, tt.wantResult, gotResult)
		})
	}
}
