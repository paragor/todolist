package models

import (
	"github.com/google/uuid"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func TestParseHumanInput(t *testing.T) {
	type args struct {
		input string
	}
	tests := []struct {
		args    args
		want    *HumanInputParserResult
		wantErr bool
	}{
		{
			args: args{
				input: " modify 358bb57b-7d84-47a0-a3d5-29fcd77f87b9  project:one bla due:2024-10-15T10:00:00 +hui blabla bla !bui notify: status:pending\t \n ",
			},
			want: &HumanInputParserResult{
				Action:     HumanActionModify,
				ActionUUID: (func(UUID string) *uuid.UUID { r := uuid.MustParse(UUID); return &r })("358bb57b-7d84-47a0-a3d5-29fcd77f87b9"),
				Options: HumanInputOptions{
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
					Status:     (func() *taskStatus { r := Pending; return &r })(),
					ExtraWords: []string{"bla", "blabla", "bla"},
				},
			},
			wantErr: false,
		},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			got, err := ParseHumanInput(tt.args.input)
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
