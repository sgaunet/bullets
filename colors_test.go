package bullets

import (
	"strings"
	"testing"
)

func TestColorize(t *testing.T) {
	result := colorize(red, "error")

	if !strings.Contains(result, "error") {
		t.Errorf("Expected colorize to contain 'error', got %q", result)
	}

	if !strings.HasPrefix(result, red) {
		t.Errorf("Expected result to start with red color code")
	}

	if !strings.HasSuffix(result, reset) {
		t.Errorf("Expected result to end with reset code")
	}
}

func TestGetBulletStyleDefault(t *testing.T) {
	customBullets := make(map[Level]string)

	tests := []struct {
		level              Level
		useSpecialBullets  bool
		expectedBullet     string
	}{
		{InfoLevel, false, bulletInfo},
		{WarnLevel, false, bulletInfo},
		{ErrorLevel, false, bulletInfo},
		{DebugLevel, false, bulletInfo}, // Debug uses bulletInfo when special bullets disabled
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			bullet, _ := getBulletStyle(tt.level, tt.useSpecialBullets, customBullets)

			if !strings.Contains(bullet, tt.expectedBullet) {
				t.Errorf("Expected bullet to contain %q, got %q", tt.expectedBullet, bullet)
			}
		})
	}
}

func TestGetBulletStyleSpecial(t *testing.T) {
	customBullets := make(map[Level]string)

	tests := []struct {
		level            Level
		expectedContains string
	}{
		{InfoLevel, bulletInfo},
		{WarnLevel, bulletWarn},
		{ErrorLevel, bulletError},
		{DebugLevel, bulletDebug},
	}

	for _, tt := range tests {
		t.Run(tt.level.String()+"_special", func(t *testing.T) {
			bullet, _ := getBulletStyle(tt.level, true, customBullets)

			if !strings.Contains(bullet, tt.expectedContains) {
				t.Errorf("Expected special bullet to contain %q, got %q", tt.expectedContains, bullet)
			}
		})
	}
}

func TestGetBulletStyleCustom(t *testing.T) {
	customBullets := map[Level]string{
		InfoLevel:  "→",
		ErrorLevel: "✖",
	}

	bullet, _ := getBulletStyle(InfoLevel, false, customBullets)
	if !strings.Contains(bullet, "→") {
		t.Errorf("Expected custom bullet '→', got %q", bullet)
	}

	bullet, _ = getBulletStyle(ErrorLevel, true, customBullets)
	if !strings.Contains(bullet, "✖") {
		t.Errorf("Expected custom bullet '✖' (should override special), got %q", bullet)
	}
}

func TestFormatMessage(t *testing.T) {
	customBullets := make(map[Level]string)

	result := formatMessage(InfoLevel, "test message", false, customBullets)

	if !strings.Contains(result, "test message") {
		t.Errorf("Expected formatted message to contain 'test message', got %q", result)
	}

	if !strings.Contains(result, bulletInfo) {
		t.Errorf("Expected formatted message to contain bullet, got %q", result)
	}
}

func TestFormatMessageWithCustomBullet(t *testing.T) {
	customBullets := map[Level]string{
		WarnLevel: "⚡",
	}

	result := formatMessage(WarnLevel, "warning", false, customBullets)

	if !strings.Contains(result, "⚡") {
		t.Errorf("Expected formatted message to contain custom bullet '⚡', got %q", result)
	}

	if !strings.Contains(result, "warning") {
		t.Errorf("Expected formatted message to contain 'warning', got %q", result)
	}
}
