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
			args:       args{name: names.Duration, text: "Duration пять"},
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

func TestBot_parseKnowledge(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		want    knowledge
		wantErr bool
	}{
		{
			name: "Link+Duration with /add command same line",
			text: `/add https://www.linkedin.com/video/event/urn:li:ugcPost:6950083329849221120/
			duration 5`,
			want: knowledge{
				link:     "https://www.linkedin.com/video/event/urn:li:ugcPost:6950083329849221120/",
				duration: 5,
			},
			wantErr: false,
		},
		{
			name: "Link+Duration /add separetely",
			text: `/add 
			https://www.linkedin.com/video/event/urn:li:ugcPost:6950083329849221120/
			ДЛИТЕЛЬНОСТЬ 14`,
			want: knowledge{
				id:            "",
				name:          "",
				adder:         "",
				knowledgeType: "",
				subtype:       "",
				theme:         "",
				sphere:        "",
				link:          "https://www.linkedin.com/video/event/urn:li:ugcPost:6950083329849221120/",
				wordCount:     0,
				duration:      14,
			},
			wantErr: false,
		},
		{
			name: "Link+Duration /add separetely",
			text: `/add 
			Ссылка: https://www.linkedin.com/video/event/urn:li:ugcPost:6950083329849221120/
			duration 16`,
			want: knowledge{
				id:            "",
				name:          "",
				adder:         "",
				knowledgeType: "",
				subtype:       "",
				theme:         "",
				sphere:        "",
				link:          "https://www.linkedin.com/video/event/urn:li:ugcPost:6950083329849221120/",
				wordCount:     0,
				duration:      16,
			},
			wantErr: false,
		},
		{
			name: "Solo Link",
			text: `https://www.youtube.com/watch?v=HGQdOX7L65o`,
			want: knowledge{
				id:            "",
				name:          "",
				adder:         "",
				knowledgeType: "",
				subtype:       "",
				theme:         "",
				sphere:        "",
				link:          "https://www.youtube.com/watch?v=HGQdOX7L65o",
				wordCount:     0,
				duration:      0,
			},
			wantErr: false,
		},
		{
			name: "Solo Link with add command",
			text: `/add https://www.linkedin.com/video/event/urn:li:ugcPost:6950083329849221120/`,
			want: knowledge{
				id:            "",
				name:          "",
				adder:         "",
				knowledgeType: "",
				subtype:       "",
				theme:         "",
				sphere:        "",
				link:          "https://www.linkedin.com/video/event/urn:li:ugcPost:6950083329849221120/",
				wordCount:     0,
				duration:      0,
			},
			wantErr: false,
		},
		{
			name: "Lots of stuff without  add command",
			text: `Ссылка: https://www.linkedin.com/video/event/urn:li:ugcPost:6950083329849221120/
			Название: Webinar: Importance of Market Research & Cognitive Design by Amazon Sr PM
			Длительность: 10
			Тема: Market Research
			Тип: Video
			Подтип: Webinar
			Сфера: PM`,
			want: knowledge{
				id:            "",
				name:          "Webinar: Importance of Market Research & Cognitive Design by Amazon Sr PM",
				adder:         "",
				knowledgeType: "Video",
				subtype:       "Webinar",
				theme:         "Market Research",
				sphere:        "PM",
				link:          "https://www.linkedin.com/video/event/urn:li:ugcPost:6950083329849221120/",
				wordCount:     0,
				duration:      10,
			},
			wantErr: false,
		},
		{
			name: "Lots of stuff with add command same line",
			text: `/add Ссылка: https://www.linkedin.com/video/event/urn:li:ugcPost:6950083329849221120/
			Название: Webinar: Importance of Market Research & Cognitive Design by Amazon Sr PM
			Длительность: 10
			Тема: Market Research
			Тип: Video
			Подтип: Webinar
			Сфера: PM`,
			want: knowledge{
				id:            "",
				name:          "Webinar: Importance of Market Research & Cognitive Design by Amazon Sr PM",
				adder:         "",
				knowledgeType: "Video",
				subtype:       "Webinar",
				theme:         "Market Research",
				sphere:        "PM",
				link:          "https://www.linkedin.com/video/event/urn:li:ugcPost:6950083329849221120/",
				wordCount:     0,
				duration:      10,
			},
			wantErr: false,
		},
		{
			name: "Lots of stuff with add command separate line",
			text: `/add 
			Ссылка: https://www.linkedin.com/video/event/urn:li:ugcPost:6950083329849221120/
			Название: Webinar: Importance of Market Research & Cognitive Design by Amazon Sr PM
			Длительность: 10
			Тема: Market Research
			Тип: Video
			Подтип: Webinar
			Сфера: PM`,
			want: knowledge{
				id:            "",
				name:          "Webinar: Importance of Market Research & Cognitive Design by Amazon Sr PM",
				adder:         "",
				knowledgeType: "Video",
				subtype:       "Webinar",
				theme:         "Market Research",
				sphere:        "PM",
				link:          "https://www.linkedin.com/video/event/urn:li:ugcPost:6950083329849221120/",
				wordCount:     0,
				duration:      10,
			},
			wantErr: false,
		},
	}
	cms, err := readCMS("./cms.json")
	if err != nil {
		t.Fatal(err)
	}
	b := Bot{t: cms}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := b.parseKnowledge(tt.text)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
