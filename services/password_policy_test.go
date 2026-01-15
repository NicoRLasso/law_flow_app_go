package services

import "testing"

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"Short password", "Short1!", true},
		{"No uppercase", "longpassword1!", true},
		{"No lowercase", "LONGPASSWORD1!", true},
		{"No number", "LongPassword!", true},
		{"No special", "LongPassword1", true},
		{"Valid password", "LongPassword1!", false},
		{"Valid password with symbols", "Secure-P@ssw0rd#", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidatePassword(tt.password); (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
