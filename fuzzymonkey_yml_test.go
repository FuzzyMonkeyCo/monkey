package main

import "testing"
import "github.com/stretchr/testify/require"

func TestReadBadVersions(t *testing.T) {
	for name, yml := range map[string][]byte{
		"empty":                     []byte(``),
		"typo in key":               []byte(`verion: 4`),
		"unsupported version":       []byte(`version: 42`),
		"bad minimum":               []byte(`version: 0`),
		"float":                     []byte(`version: 4.2`),
		"string a la docker config": []byte(`version: '4'`),
		"missing value":             []byte(`version: `),
		"typo in value":             []byte(`version:\n1`),
		"duplicate key":             []byte(`{version: 1, port: 80, port: 443}`),
	} {
		t.Run(name, func(t *testing.T) {
			cfg, err := newCfg(yml, false)
			require.Error(t, err)
			require.Equal(t, (*ymlCfg)(nil), cfg)
		})
	}
}

func TestV1ReadErrors(t *testing.T) {
	for name, yml := range map[string][]byte{
		"duplicate key": []byte(`
version: 1
documentation:
  kind: OpenAPIv3
  file: some_file.json
documentation: blbl
`),
		"unexpected key": []byte(`
version: 1
documentation:
  kind: OpenAPIv3
  file: some_file.json
blabla: blbl
`),
		"more than one issue at once": []byte(`
version: 1
documentation: yes
documentation:
  kind: OpenAPIv3
  file: some_file.json
blabla: blbl
`),
	} {
		t.Run(name, func(t *testing.T) {
			cfg, err := newCfg(yml, false)
			require.Error(t, err)
			require.Equal(t, (*ymlCfg)(nil), cfg)
		})
	}
}

func TestV1ReadAllSet(t *testing.T) {
	yml := []byte(`
version: 1
documentation:
  host: app.vcap.me
  port: 8000
  file: ./spec.yml
  kind: OpenAPIv3
start:
- make service-start
reset:
- make service-restart
stop:
- make service-kill
`)

	cfg, err := newCfg(yml, false)
	require.NoError(t, err)
	require.Equal(t, "app.vcap.me", cfg.Host)
	require.Equal(t, "8000", cfg.Port)
	require.Equal(t, "./spec.yml", cfg.File)
	require.Equal(t, "OpenAPIv3", cfg.Kind)
	require.Equal(t, []string{"make service-start"}, cfg.Start)
	require.Equal(t, []string{"make service-restart"}, cfg.Reset)
	require.Equal(t, []string{"make service-kill"}, cfg.Stop)
}

func TestV1ReadDefaults(t *testing.T) {
	for name, yml := range map[string][]byte{
		"version only": []byte("version: 1"),
		"version & host": []byte(`
version: 1
documentation:
  host: localhost
`),
		"version & port": []byte(`
version: 1
documentation:
  port: 3000
`),
	} {
		t.Run(name, func(t *testing.T) {
			cfg, err := newCfg(yml, false)
			require.NoError(t, err)
			require.Equal(t, "localhost", cfg.Host)
			require.Equal(t, "3000", cfg.Port)
		})
	}
}
