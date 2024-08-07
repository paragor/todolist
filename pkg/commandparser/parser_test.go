package commandparser

import (
	"reflect"
	"strconv"
	"testing"
	"time"
)

func TestParseCommand(t *testing.T) {
	type args struct {
		command string
	}
	tests := []struct {
		args    args
		want    *ParserResult
		wantErr bool
	}{
		{
			args: args{
				command: "list   project:one bla due:2024-10-15T10:00:00 +hui blabla bla -bui notify: status:pending",
			},
			want: &ParserResult{
				Action:  "list",
				Options: map[string]string{},
				Project: AddOrDeleteValue[string]{
					IsExists: true,
					IsAdd:    true,
					Value:    "one",
				},
				Tags: []AddOrDeleteValue[string]{
					{
						IsExists: true,
						IsAdd:    true,
						Value:    "hui",
					},
					{
						IsExists: true,
						IsAdd:    false,
						Value:    "bui",
					},
				},
				Notify: AddOrDeleteValue[time.Time]{
					IsExists: true,
					IsAdd:    false,
					Value:    time.Time{},
				},
				Due: AddOrDeleteValue[time.Time]{
					IsExists: true,
					IsAdd:    true,
					Value:    time.Date(2024, 10, 15, 10, 0, 0, 0, time.Local),
				},
				Status:     (func() *string { r := "pending"; return &r })(),
				ExtraWords: []string{"bla", "blabla", "bla"},
			},
			wantErr: false,
		},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			got, err := ParseCommand(tt.args.command)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseCommand() got = \n%+v, want \n%+v", got, tt.want)
			}
		})
	}
}
