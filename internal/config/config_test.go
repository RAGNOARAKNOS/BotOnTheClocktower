package config

import "testing"

func TestLoadRequiresToken(t *testing.T) {
	t.Setenv("BOTAPIKEY", "")
	if _, err := Load(); err == nil {
		t.Error("Load() with no token returned no error")
	}

	t.Setenv("BOTAPIKEY", "secret")
	if cfg, err := Load(); err != nil || cfg.Token != "secret" {
		t.Errorf("Load() = %+v, %v; want the token and no error", cfg, err)
	}
}
