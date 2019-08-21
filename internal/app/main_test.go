package app

import "testing"

func Test_parseScriptOutputLine(t *testing.T) {
	type args struct {
		line string
	}
	tests := []struct {
		name      string
		args      args
		wantName  string
		wantValue float64
		wantErr   bool
	}{
		{"success1", args{"value1: 1.234"}, "value1", 1.234, false},
		{"success2", args{"value2: 42"}, "value2", 42, false},
		{"invalid name", args{"invalid-name: 61.2"}, "", 0, true},
		{"invalid value", args{"value3: 1.abc"}, "value3", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotValue, err := parseScriptOutputLine(tt.args.line)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseScriptOutputLine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotName != tt.wantName {
				t.Errorf("parseScriptOutputLine() gotName = %v, want %v", gotName, tt.wantName)
			}
			if gotValue != tt.wantValue {
				t.Errorf("parseScriptOutputLine() gotValue = %v, want %v", gotValue, tt.wantValue)
			}
		})
	}
}
