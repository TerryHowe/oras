package resource

import "testing"

func Test_parseOCILayoutReference(t *testing.T) {
	opts := OciLayout{
		RawReference: "/test",
	}
	tests := []struct {
		name    string
		raw     string
		want    string
		want1   string
		wantErr bool
	}{
		{"Empty input", "", "", "", true},
		{"Empty path and tag", ":", "", "", true},
		{"Empty path and digest", "@", "", "", false},
		{"Empty digest", "path@", "path", "", false},
		{"Empty tag", "path:", "path", "", false},
		{"path and digest", "path@digest", "path", "digest", false},
		{"path and tag", "path:tag", "path", "tag", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts.RawReference = tt.raw
			err := opts.Parse()
			if (err != nil) != tt.wantErr {
				t.Errorf("parseOCILayoutReference() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if opts.Path != tt.want {
				t.Errorf("parseOCILayoutReference() got = %v, want %v", opts.Path, tt.want)
			}
			if opts.Reference != tt.want1 {
				t.Errorf("parseOCILayoutReference() got1 = %v, want %v", opts.Reference, tt.want1)
			}
		})
	}
}
